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

// IUserFeedSeenRepository is a mock type for the repository.IUserFeedSeenRepository type
type IUserFeedSeenRepository struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, userID, feed
func (_m *IUserFeedSeenRepository) Get(ctx context.Context, userID int64, feed string) (*model.UserFeedSeen, error) {
	ret := _m.Called(ctx, userID, feed)

	if rf, ok := ret.Get(0).(func(context.Context, int64, string) (*model.UserFeedSeen, error)); ok {
		return rf(ctx, userID, feed)
	}

	var r0 *model.UserFeedSeen
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.UserFeedSeen)
	}
	return r0, ret.Error(1)
}

// Set provides a mock function with given fields: ctx, userID, feed, seenAt
func (_m *IUserFeedSeenRepository) Set(ctx context.Context, userID int64, feed string, seenAt time.Time) error {
	ret := _m.Called(ctx, userID, feed, seenAt)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, string, time.Time) error); ok {
		r0 = rf(ctx, userID, feed, seenAt)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIUserFeedSeenRepository creates a new instance of IUserFeedSeenRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIUserFeedSeenRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *IUserFeedSeenRepository {
	m := &IUserFeedSeenRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
