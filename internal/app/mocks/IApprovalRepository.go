// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"
	"time"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// IApprovalRepository is a mock type for the repository.IApprovalRepository type
type IApprovalRepository struct {
	mock.Mock
}

// GetHeaderByTaskID provides a mock function with given fields: ctx, taskID
func (_m *IApprovalRepository) GetHeaderByTaskID(ctx context.Context, taskID string) (*model.ApprovalHeader, error) {
	ret := _m.Called(ctx, taskID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.ApprovalHeader, error)); ok {
		return rf(ctx, taskID)
	}

	var r0 *model.ApprovalHeader
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalHeader)
	}
	return r0, ret.Error(1)
}

// GetHeaderByID provides a mock function with given fields: ctx, id
func (_m *IApprovalRepository) GetHeaderByID(ctx context.Context, id uint) (*model.ApprovalHeader, error) {
	ret := _m.Called(ctx, id)

	if rf, ok := ret.Get(0).(func(context.Context, uint) (*model.ApprovalHeader, error)); ok {
		return rf(ctx, id)
	}

	var r0 *model.ApprovalHeader
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalHeader)
	}
	return r0, ret.Error(1)
}

// GetDetailByID provides a mock function with given fields: ctx, id
func (_m *IApprovalRepository) GetDetailByID(ctx context.Context, id uint) (*model.ApprovalDetail, error) {
	ret := _m.Called(ctx, id)

	if rf, ok := ret.Get(0).(func(context.Context, uint) (*model.ApprovalDetail, error)); ok {
		return rf(ctx, id)
	}

	var r0 *model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalDetail)
	}
	return r0, ret.Error(1)
}

// GetDetailsByHeaderID provides a mock function with given fields: ctx, headerID
func (_m *IApprovalRepository) GetDetailsByHeaderID(ctx context.Context, headerID uint) ([]*model.ApprovalDetail, error) {
	ret := _m.Called(ctx, headerID)

	if rf, ok := ret.Get(0).(func(context.Context, uint) ([]*model.ApprovalDetail, error)); ok {
		return rf(ctx, headerID)
	}

	var r0 []*model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalDetail)
	}
	return r0, ret.Error(1)
}

// GetPendingByApprover provides a mock function with given fields: ctx, approverID, search, orderBy, limit, offset
func (_m *IApprovalRepository) GetPendingByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) ([]*model.ApprovalDetail, int64, error) {
	ret := _m.Called(ctx, approverID, search, orderBy, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, int64, string, string, int, int) ([]*model.ApprovalDetail, int64, error)); ok {
		return rf(ctx, approverID, search, orderBy, limit, offset)
	}

	var r0 []*model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalDetail)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetActedByApprover provides a mock function with given fields: ctx, approverID, search, orderBy, limit, offset
func (_m *IApprovalRepository) GetActedByApprover(ctx context.Context, approverID int64, search string, orderBy string, limit int, offset int) ([]*model.ApprovalDetail, int64, error) {
	ret := _m.Called(ctx, approverID, search, orderBy, limit, offset)

	if rf, ok := ret.Get(0).(func(context.Context, int64, string, string, int, int) ([]*model.ApprovalDetail, int64, error)); ok {
		return rf(ctx, approverID, search, orderBy, limit, offset)
	}

	var r0 []*model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.ApprovalDetail)
	}
	return r0, ret.Get(1).(int64), ret.Error(2)
}

// GetPendingDetailByApprover provides a mock function with given fields: ctx, headerID, approverID
func (_m *IApprovalRepository) GetPendingDetailByApprover(ctx context.Context, headerID uint, approverID int64) (*model.ApprovalDetail, error) {
	ret := _m.Called(ctx, headerID, approverID)

	if rf, ok := ret.Get(0).(func(context.Context, uint, int64) (*model.ApprovalDetail, error)); ok {
		return rf(ctx, headerID, approverID)
	}

	var r0 *model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalDetail)
	}
	return r0, ret.Error(1)
}

// GetHeldDetailByApprover provides a mock function with given fields: ctx, headerID, approverID
func (_m *IApprovalRepository) GetHeldDetailByApprover(ctx context.Context, headerID uint, approverID int64) (*model.ApprovalDetail, error) {
	ret := _m.Called(ctx, headerID, approverID)

	if rf, ok := ret.Get(0).(func(context.Context, uint, int64) (*model.ApprovalDetail, error)); ok {
		return rf(ctx, headerID, approverID)
	}

	var r0 *model.ApprovalDetail
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.ApprovalDetail)
	}
	return r0, ret.Error(1)
}

// UpdateDetailIfHeld provides a mock function with given fields: ctx, id, updates
func (_m *IApprovalRepository) UpdateDetailIfHeld(ctx context.Context, id uint, updates map[string]interface{}) (int64, error) {
	ret := _m.Called(ctx, id, updates)

	if rf, ok := ret.Get(0).(func(context.Context, uint, map[string]interface{}) (int64, error)); ok {
		return rf(ctx, id, updates)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// CreateHeader provides a mock function with given fields: ctx, header
func (_m *IApprovalRepository) CreateHeader(ctx context.Context, header *model.ApprovalHeader) error {
	ret := _m.Called(ctx, header)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.ApprovalHeader) error); ok {
		r0 = rf(ctx, header)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateDetails provides a mock function with given fields: ctx, details
func (_m *IApprovalRepository) CreateDetails(ctx context.Context, details []*model.ApprovalDetail) error {
	ret := _m.Called(ctx, details)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.ApprovalDetail) error); ok {
		r0 = rf(ctx, details)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateHeader provides a mock function with given fields: ctx, id, updates
func (_m *IApprovalRepository) UpdateHeader(ctx context.Context, id uint, updates map[string]interface{}) error {
	ret := _m.Called(ctx, id, updates)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, map[string]interface{}) error); ok {
		r0 = rf(ctx, id, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDetailByID provides a mock function with given fields: ctx, id, updates
func (_m *IApprovalRepository) UpdateDetailByID(ctx context.Context, id uint, updates map[string]interface{}) error {
	ret := _m.Called(ctx, id, updates)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, uint, map[string]interface{}) error); ok {
		r0 = rf(ctx, id, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDetailIfPending provides a mock function with given fields: ctx, id, updates
func (_m *IApprovalRepository) UpdateDetailIfPending(ctx context.Context, id uint, updates map[string]interface{}) (int64, error) {
	ret := _m.Called(ctx, id, updates)

	if rf, ok := ret.Get(0).(func(context.Context, uint, map[string]interface{}) (int64, error)); ok {
		return rf(ctx, id, updates)
	}

	return ret.Get(0).(int64), ret.Error(1)
}

// GetDecisionRolesByTaskIDs provides a mock function with given fields: ctx, taskIDs, statuses
func (_m *IApprovalRepository) GetDecisionRolesByTaskIDs(ctx context.Context, taskIDs []string, statuses []string) ([]model.TaskDecisionRole, error) {
	ret := _m.Called(ctx, taskIDs, statuses)

	if rf, ok := ret.Get(0).(func(context.Context, []string, []string) ([]model.TaskDecisionRole, error)); ok {
		return rf(ctx, taskIDs, statuses)
	}

	var r0 []model.TaskDecisionRole
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.TaskDecisionRole)
	}
	return r0, ret.Error(1)
}

// GetNewPendingByApprover provides a mock function with given fields: ctx, approverID, since
func (_m *IApprovalRepository) GetNewPendingByApprover(ctx context.Context, approverID int64, since time.Time) ([]model.ApproverNewsRow, error) {
	ret := _m.Called(ctx, approverID, since)

	if rf, ok := ret.Get(0).(func(context.Context, int64, time.Time) ([]model.ApproverNewsRow, error)); ok {
		return rf(ctx, approverID, since)
	}

	var r0 []model.ApproverNewsRow
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.ApproverNewsRow)
	}
	return r0, ret.Error(1)
}

// GetDivHeadReturnsAfterApproval provides a mock function with given fields: ctx, approverID, since
func (_m *IApprovalRepository) GetDivHeadReturnsAfterApproval(ctx context.Context, approverID int64, since time.Time) ([]model.ApproverNewsRow, error) {
	ret := _m.Called(ctx, approverID, since)

	if rf, ok := ret.Get(0).(func(context.Context, int64, time.Time) ([]model.ApproverNewsRow, error)); ok {
		return rf(ctx, approverID, since)
	}

	var r0 []model.ApproverNewsRow
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.ApproverNewsRow)
	}
	return r0, ret.Error(1)
}

// GetFirstSubmittedAt provides a mock function with given fields: ctx, taskIDs
func (_m *IApprovalRepository) GetFirstSubmittedAt(ctx context.Context, taskIDs []string) ([]model.TaskSubmittedAt, error) {
	ret := _m.Called(ctx, taskIDs)

	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]model.TaskSubmittedAt, error)); ok {
		return rf(ctx, taskIDs)
	}

	var r0 []model.TaskSubmittedAt
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.TaskSubmittedAt)
	}
	return r0, ret.Error(1)
}

// GetLevelNotesByTaskIDs provides a mock function with given fields: ctx, taskIDs, status
func (_m *IApprovalRepository) GetLevelNotesByTaskIDs(ctx context.Context, taskIDs []string, status string) ([]model.TaskLevelNote, error) {
	ret := _m.Called(ctx, taskIDs, status)

	if rf, ok := ret.Get(0).(func(context.Context, []string, string) ([]model.TaskLevelNote, error)); ok {
		return rf(ctx, taskIDs, status)
	}

	var r0 []model.TaskLevelNote
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]model.TaskLevelNote)
	}
	return r0, ret.Error(1)
}

// NewIApprovalRepository creates a new instance of IApprovalRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIApprovalRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IApprovalRepository {
	m := &IApprovalRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
