// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ISiteRepository is a mock type for the repository.ISiteRepository type
type ISiteRepository struct {
	mock.Mock
}

// GetByIDs provides a mock function with given fields: ctx, siteIDs
func (_m *ISiteRepository) GetByIDs(ctx context.Context, siteIDs []string) ([]*model.Site, error) {
	ret := _m.Called(ctx, siteIDs)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]*model.Site, error)); ok {
		return rf(ctx, siteIDs)
	}

	var r0 []*model.Site
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Site)
	}
	return r0, ret.Error(1)
}

// GetIDsBySearch provides a mock function with given fields: ctx, search
func (_m *ISiteRepository) GetIDsBySearch(ctx context.Context, search string) ([]string, error) {
	ret := _m.Called(ctx, search)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]string, error)); ok {
		return rf(ctx, search)
	}

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}
	return r0, ret.Error(1)
}

// GetByID provides a mock function with given fields: ctx, siteID
func (_m *ISiteRepository) GetByID(ctx context.Context, siteID string) (*model.Site, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.Site, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 *model.Site
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.Site)
	}
	return r0, ret.Error(1)
}

// GetAttachmentsBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ISiteRepository) GetAttachmentsBySiteID(ctx context.Context, siteID string) ([]*model.SiteAttachment, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*model.SiteAttachment, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 []*model.SiteAttachment
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAttachment)
	}
	return r0, ret.Error(1)
}

// Get provides a mock function with given fields: ctx, offset, limit, orderBy, city, region
func (_m *ISiteRepository) Get(ctx context.Context, offset int, limit int, orderBy string, city string, region string) ([]*model.Site, int64, error) {
	ret := _m.Called(ctx, offset, limit, orderBy, city, region)

	if rf, ok := ret.Get(0).(func(context.Context, int, int, string, string, string) ([]*model.Site, int64, error)); ok {
		return rf(ctx, offset, limit, orderBy, city, region)
	}

	var r0 []*model.Site
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Site)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetAll provides a mock function with given fields: ctx
func (_m *ISiteRepository) GetAll(ctx context.Context) ([]*model.Site, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]*model.Site, error)); ok {
		return rf(ctx)
	}

	var r0 []*model.Site
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.Site)
	}
	return r0, ret.Error(1)
}

// GetActiveStoreMaps provides a mock function with given fields: ctx
func (_m *ISiteRepository) GetActiveStoreMaps(ctx context.Context) ([]model.StoreMap, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]model.StoreMap, error)); ok {
		return rf(ctx)
	}

	var r0 []model.StoreMap
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.StoreMap)
	}
	return r0, ret.Error(1)
}

// GetActiveStores provides a mock function with given fields: ctx, search, limit, offset
func (_m *ISiteRepository) GetActiveStores(ctx context.Context, search string, limit int, offset int) ([]model.StoreMap, int64, error) {
	ret := _m.Called(ctx, search, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, string, int, int) ([]model.StoreMap, int64, error)); ok {
		return rf(ctx, search, limit, offset)
	}

	var r0 []model.StoreMap
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.StoreMap)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetAttachmentByID provides a mock function with given fields: ctx, id
func (_m *ISiteRepository) GetAttachmentByID(ctx context.Context, id uint) (*model.SiteAttachment, error) {
	ret := _m.Called(ctx, id)

	if rf, ok := ret.Get(0).(func(context.Context, uint) (*model.SiteAttachment, error)); ok {
		return rf(ctx, id)
	}

	var r0 *model.SiteAttachment
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.SiteAttachment)
	}
	return r0, ret.Error(1)
}

// GetByName1 provides a mock function with given fields: ctx, name1
func (_m *ISiteRepository) GetByName1(ctx context.Context, name1 string) (*model.Site, error) {
	ret := _m.Called(ctx, name1)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.Site, error)); ok {
		return rf(ctx, name1)
	}

	var r0 *model.Site
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.Site)
	}
	return r0, ret.Error(1)
}

// Upsert provides a mock function with given fields: ctx, site
func (_m *ISiteRepository) Upsert(ctx context.Context, site *model.Site) error {
	ret := _m.Called(ctx, site)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Site) error); ok {
		r0 = rf(ctx, site)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReplaceAttachments provides a mock function with given fields: ctx, siteID, attachments
func (_m *ISiteRepository) ReplaceAttachments(ctx context.Context, siteID string, attachments []*model.SiteAttachment) error {
	ret := _m.Called(ctx, siteID, attachments)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []*model.SiteAttachment) error); ok {
		r0 = rf(ctx, siteID, attachments)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetActiveStorePoints provides a mock function with given fields: ctx
func (_m *ISiteRepository) GetActiveStorePoints(ctx context.Context) ([]model.StoreMap, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]model.StoreMap, error)); ok {
		return rf(ctx)
	}

	var r0 []model.StoreMap
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.StoreMap)
	}
	return r0, ret.Error(1)
}

// NewISiteRepository creates a new instance of ISiteRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteRepository {
	m := &ISiteRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
