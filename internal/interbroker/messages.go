package interbroker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/jorgebay/soda/internal/conf"
	"github.com/jorgebay/soda/internal/types"
)

type opcode uint8
type streamId uint16

const (
	startupOp opcode = iota
	readyOp
	errorOp
	dataOp
	dataResponseOp
)

// header is the interbroker message header
type header struct {
	// Strict ordering, exported fields
	Version    uint8
	StreamId   streamId
	Op         opcode
	BodyLength uint32
	// TODO: Add CRC to header
}

const headerSize = 1 + // version
	2 + // stream id
	1 + // op
	4 // length

type dataRequestMeta struct {
	// Strict ordering, exported fields
	SegmentId int64
	Token     types.Token
	GenId     uint32
	// TopicLength is the size in bytes of the topic name
	TopicLength uint8
}

const dataRequestMetaSize = 8 + // segment id
	8 + // token
	2 + // genId
	1 // topicLength

type dataRequest struct {
	meta  dataRequestMeta
	topic string
	data  []byte
	// response from replica
	response chan dataResponse
	// result from append as a replica
	appendResult chan error
	ctxt         context.Context
}

func (r *dataRequest) BodyLength() uint32 {
	return uint32(dataRequestMetaSize) + uint32(r.meta.TopicLength) + uint32(len(r.data))
}

func (r *dataRequest) DataBlock() []byte {
	return r.data
}

func (r *dataRequest) Replication() *types.ReplicationInfo {
	return nil
}

func (r *dataRequest) SegmentId() int64 {
	return r.meta.SegmentId
}

func (r *dataRequest) SetResult(err error) {
	r.appendResult <- err
}

func (r *dataRequest) Marshal(w types.StringWriter, header *header) {
	writeHeader(w, header)
	binary.Write(w, conf.Endianness, r.meta)
	w.WriteString(r.topic)
	w.Write(r.data)
}

func (r *dataRequest) topicId() types.TopicDataId {
	return types.TopicDataId{
		Name:  r.topic,
		Token: r.meta.Token,
		GenId: r.meta.GenId,
	}
}

type dataResponse interface {
	Marshal(w io.Writer) error
}

type errorResponse struct {
	message  string
	streamId streamId
}

func newErrorResponse(message string, requestHeader *header) *errorResponse {
	return &errorResponse{message: message, streamId: requestHeader.StreamId}
}

// Deserializes an error response into a buffer
func (r *errorResponse) Marshal(w io.Writer) error {
	message := []byte(r.message)
	if err := writeHeader(w, &header{
		Version:    1,
		StreamId:   r.streamId,
		Op:         errorOp,
		BodyLength: uint32(len(message)),
	}); err != nil {
		return err
	}

	_, err := w.Write(message)
	return err
}

func writeHeader(w io.Writer, header *header) error {
	return binary.Write(w, conf.Endianness, header)
}

func readHeader(buffer []byte) (*header, error) {
	header := &header{}
	err := binary.Read(bytes.NewReader(buffer), conf.Endianness, header)
	return header, err
}

type emptyResponse struct {
	streamId streamId
	op       opcode
}

func (r *emptyResponse) Marshal(w io.Writer) error {
	return writeHeader(w, &header{
		Version:    1,
		StreamId:   r.streamId,
		Op:         r.op,
		BodyLength: 0,
	})
}

func unmarshallResponse(header *header, body []byte) dataResponse {
	if header.Op == errorOp {
		return newErrorResponse(string(body), header)
	}
	return &emptyResponse{
		streamId: header.StreamId,
		op:       header.Op,
	}
}

// decodes into a data request (without response channel)
func unmarshalDataRequest(body []byte) (*dataRequest, error) {
	meta := dataRequestMeta{}
	reader := bytes.NewReader(body)
	if err := binary.Read(reader, conf.Endianness, &meta); err != nil {
		return nil, err
	}

	index := dataRequestMetaSize
	topic := string(body[index : index+int(meta.TopicLength)])
	index += int(meta.TopicLength)
	request := &dataRequest{
		meta:  meta,
		topic: topic,
		data:  body[index:],
	}

	return request, nil
}
