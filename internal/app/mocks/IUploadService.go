// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"
	"io"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IUploadService is a mock type for the service.IUploadService type
type IUploadService struct {
	mock.Mock
}

// Add provides a mock function with given fields: ctx, fileName, content
func (_m *IUploadService) Add(ctx context.Context, fileName string, content io.Reader) (dto.UploadResponse, error) {
	ret := _m.Called(ctx, fileName, content)

	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader) (dto.UploadResponse, error)); ok {
		return rf(ctx, fileName, content)
	}

	return ret.Get(0).(dto.UploadResponse), ret.Error(1)
}

// Remove provides a mock function with given fields: ctx, path
func (_m *IUploadService) Remove(ctx context.Context, path string) error {
	ret := _m.Called(ctx, path)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// OpenAttachment provides a mock function with given fields: ctx, attachmentID
func (_m *IUploadService) OpenAttachment(ctx context.Context, attachmentID uint) (*model.SiteAttachment, io.ReadCloser, error) {
	ret := _m.Called(ctx, attachmentID)

	if rf, ok := ret.Get(0).(func(context.Context, uint) (*model.SiteAttachment, io.ReadCloser, error)); ok {
		return rf(ctx, attachmentID)
	}

	var r0 *model.SiteAttachment
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.SiteAttachment)
	}
	return r0, ret.Get(1).(io.ReadCloser), ret.Error(2)
}

// NewIUploadService creates a new instance of IUploadService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIUploadService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IUploadService {
	m := &IUploadService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
