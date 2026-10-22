// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// ICompetitorBrandService is a mock type for the service.ICompetitorBrandService type
type ICompetitorBrandService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx
func (_m *ICompetitorBrandService) Get(ctx context.Context) ([]dto.CompetitorBrandResponseGet, error) {
	ret := _m.Called(ctx)

	if rf, ok := ret.Get(0).(func(context.Context) ([]dto.CompetitorBrandResponseGet, error)); ok {
		return rf(ctx)
	}

	var r0 []dto.CompetitorBrandResponseGet
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]dto.CompetitorBrandResponseGet)
	}
	return r0, ret.Error(1)
}

// NewICompetitorBrandService creates a new instance of ICompetitorBrandService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewICompetitorBrandService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ICompetitorBrandService {
	m := &ICompetitorBrandService{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
