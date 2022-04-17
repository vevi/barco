package data

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/barcostreams/barco/internal/conf"
	. "github.com/barcostreams/barco/internal/types"
	"github.com/barcostreams/barco/internal/utils"
	"github.com/rs/zerolog/log"
)

const (
	DirectoryPermissions os.FileMode = 0755
	FilePermissions      os.FileMode = 0644
)

const streamBufferLength = 2 // Amount of buffers for file streaming

type Datalog interface {
	Initializer

	// Blocks until there's an available buffer to be used to stream.
	// After use, it should be released
	StreamBuffer() *bytes.Buffer

	// Releases the stream buffer
	ReleaseStreamBuffer(b *bytes.Buffer)
}

func NewDatalog(config conf.DatalogConfig) Datalog {
	streamBufferChan := make(chan *bytes.Buffer, 2)
	// Add 2 buffers by default with 1/16 the capacity
	for i := 0; i < streamBufferLength; i++ {
		streamBufferChan <- utils.NewBufferCap(config.StreamBufferSize() / 16)
	}

	return &datalog{
		config:           config,
		streamBufferChan: streamBufferChan,
	}
}

type datalog struct {
	config           conf.DatalogConfig
	streamBufferChan chan *bytes.Buffer
}

func (d *datalog) Init() error {
	return nil
}

func (d *datalog) StreamBuffer() *bytes.Buffer {
	return <-d.streamBufferChan
}

func (d *datalog) ReleaseStreamBuffer(b *bytes.Buffer) {
	d.streamBufferChan <- b
}

// Seeks the position and fills the buffer with chunks until maxSize or maxRecords is reached.
// Opens and close the file handle. It may issue several reads to reach to the position.
func (d *datalog) readFileFrom(
	buf []byte,
	maxSize int,
	segmentId int64,
	startOffset int64,
	maxRecords int,
	topic *TopicDataId,
) ([]byte, error) {
	basePath := d.config.DatalogPath(topic)
	fileOffset := tryReadIndexFile(basePath, fmt.Sprint(segmentId), startOffset)
	fileName := conf.SegmentFileName(segmentId)

	if maxSize < len(buf) {
		buf = buf[:maxSize]
	}

	file, err := os.OpenFile(filepath.Join(basePath, fileName), conf.SegmentFileReadFlags, 0)
	if err != nil {
		log.Err(err).Msgf("Could not open file %s/%s", basePath, fileName)
		return nil, err
	}

	if fileOffset > 0 {
		if _, err := file.Seek(fileOffset, io.SeekStart); err != nil {
			log.Err(err).Msgf("Could not seek position in file %s/%s", basePath, fileName)
			return nil, err
		}
	}

	remainderIndex := 0
	//read chunks until a segment containing startOffset is found
	for {
		n, err := file.Read(alignBuffer(buf[remainderIndex:]))
		if err != nil && err != io.EOF {
			log.Err(err).Msgf("Could not read file %s/%s", basePath, fileName)
			return nil, err
		}

		totalRead := remainderIndex + n
		if totalRead < chunkHeaderSize {
			return nil, nil
		}

		remainderIndex = 0
		if chunksBuf, completeChunk, err := readChunksUntil(buf[:totalRead], startOffset, maxRecords); err != nil {
			log.Err(err).Msgf("Error reading chunks in file %s/%s", basePath, fileName)
			return nil, err
		} else if completeChunk {
			return chunksBuf, nil
		} else {
			// Partial chunk
			copy(buf, chunksBuf)
			remainderIndex = len(chunksBuf)
		}

		if err == io.EOF {
			return nil, nil
		}
	}
}

// Returns a slice of the given buffer containing the chunks when completed is true.
// When completed is false, it returns the remaining
func readChunksUntil(buf []byte, startOffset int64, maxRecords int) ([]byte, bool, error) {
	headerBuf := make([]byte, chunkHeaderSize)
	for len(buf) > 0 {
		header, alignment, err := readNextChunk(buf, headerBuf)
		fmt.Println("--Read chunk in loop", header, err)
		if err != nil {
			return nil, false, err
		}

		if header == nil {
			// Incomplete chunk
			return buf, false, nil
		}

		if startOffset >= header.Start && startOffset < header.Start+int64(header.RecordLength) {
			// We found the starting chunk
			end := 0
			maxOffset := startOffset + int64(maxRecords) - 1
			for {
				end += alignment + chunkHeaderSize + int(header.BodyLength)
				header, alignment, _ = readNextChunk(buf[end:], headerBuf)
				if header == nil || header.Start > maxOffset {
					// Either there is no next chunk in buffer
					// Or the next chunk is not needed to be returned
					break
				}
			}
			return buf[:end], true, nil
		}

		// Skip the chunk
		buf = buf[alignment+chunkHeaderSize+int(header.BodyLength):]
	}
	return nil, false, nil
}

// Returns a header when it's contained in buf, otherwise it returns nil
func readNextChunk(buf []byte, headerBuf []byte) (*chunkHeader, int, error) {
	alignment := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] != alignmentFlag {
			break
		}
		alignment++
	}

	buf = buf[alignment:]

	if chunkHeaderSize > len(buf) {
		// Incomplete header
		return nil, 0, nil
	}

	// Read through the buffer
	header, err := readChunkHeader(bytes.NewReader(buf), headerBuf)
	if err != nil {
		return nil, 0, err
	}

	if int(header.BodyLength)+chunkHeaderSize > len(buf) {
		// Incomplete chunk
		return nil, 0, nil
	}

	return header, alignment, nil
}

func alignBuffer(buf []byte) []byte {
	bytesToAlign := len(buf) % alignmentSize
	if bytesToAlign > 0 {
		// Crop the last bytes to make to compatible with DIRECT I/O
		buf = buf[:len(buf)-bytesToAlign]
	}
	return buf
}
