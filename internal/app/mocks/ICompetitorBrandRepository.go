// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ICompetitorBrandRepository is a mock type for the repository.ICompetitorBrandRepository type
type ICompetitorBrandRepository struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx
func (_m *ICompetitorBrandRepository) Get(ctx context.Context) ([]*model.CompetitorBrand, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]*model.CompetitorBrand, error)); ok {
		return rf(ctx)
	}

	var r0 []*model.CompetitorBrand
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.CompetitorBrand)
	}
	return r0, ret.Error(1)
}

// NewICompetitorBrandRepository creates a new instance of ICompetitorBrandRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewICompetitorBrandRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ICompetitorBrandRepository {
	m := &ICompetitorBrandRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
