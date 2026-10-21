// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"
)

// IStorageClient is a mock type for the client.IStorageClient type
type IStorageClient struct {
	mock.Mock
}

// Write provides a mock function with given fields: ctx, bucket, object, content
func (_m *IStorageClient) Write(ctx context.Context, bucket string, object string, content io.Reader) error {
	ret := _m.Called(ctx, bucket, object, content)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, io.Reader) error); ok {
		r0 = rf(ctx, bucket, object, content)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Open provides a mock function with given fields: ctx, bucket, object
func (_m *IStorageClient) Open(ctx context.Context, bucket string, object string) (io.ReadCloser, error) {
	ret := _m.Called(ctx, bucket, object)

	if rf, ok := ret.Get(0).(func(context.Context, string, string) (io.ReadCloser, error)); ok {
		return rf(ctx, bucket, object)
	}

	return ret.Get(0).(io.ReadCloser), ret.Error(1)
}

// Delete provides a mock function with given fields: ctx, bucket, object
func (_m *IStorageClient) Delete(ctx context.Context, bucket string, object string) error {
	ret := _m.Called(ctx, bucket, object)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, bucket, object)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIStorageClient creates a new instance of IStorageClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIStorageClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IStorageClient {
	m := &IStorageClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
