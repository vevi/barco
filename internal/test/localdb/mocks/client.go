// Code generated by mockery 2.9.0. DO NOT EDIT.

package mocks

import (
	types "github.com/jorgebay/soda/internal/types"
	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *Client) Close() {
	_m.Called()
}

// CommitGeneration provides a mock function with given fields: generation
func (_m *Client) CommitGeneration(generation *types.Generation) error {
	ret := _m.Called(generation)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Generation) error); ok {
		r0 = rf(generation)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DbWasNewlyCreated provides a mock function with given fields:
func (_m *Client) DbWasNewlyCreated() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// GetGenerationsByToken provides a mock function with given fields: token
func (_m *Client) GetGenerationsByToken(token types.Token) ([]types.Generation, error) {
	ret := _m.Called(token)

	var r0 []types.Generation
	if rf, ok := ret.Get(0).(func(types.Token) []types.Generation); ok {
		r0 = rf(token)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Generation)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(types.Token) error); ok {
		r1 = rf(token)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Init provides a mock function with given fields:
func (_m *Client) Init() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SaveOffset provides a mock function with given fields: group, topic, token, rangeIndex, value
func (_m *Client) SaveOffset(group string, topic string, token types.Token, rangeIndex types.RangeIndex, value types.Offset) error {
	ret := _m.Called(group, topic, token, rangeIndex, value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, types.Token, types.RangeIndex, types.Offset) error); ok {
		r0 = rf(group, topic, token, rangeIndex, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
