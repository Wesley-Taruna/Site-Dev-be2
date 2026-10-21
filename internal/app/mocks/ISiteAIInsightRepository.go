// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/mock"
)

// ISiteAIInsightRepository is a mock type for the repository.ISiteAIInsightRepository type
type ISiteAIInsightRepository struct {
	mock.Mock
}

// Replace provides a mock function with given fields: ctx, insight
func (_m *ISiteAIInsightRepository) Replace(ctx context.Context, insight *model.SiteAIInsight) error {
	ret := _m.Called(ctx, insight)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.SiteAIInsight) error); ok {
		r0 = rf(ctx, insight)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetLatestBySiteID provides a mock function with given fields: ctx, siteID
func (_m *ISiteAIInsightRepository) GetLatestBySiteID(ctx context.Context, siteID string) (*model.SiteAIInsight, error) {
	ret := _m.Called(ctx, siteID)

	if rf, ok := ret.Get(0).(func(context.Context, string) (*model.SiteAIInsight, error)); ok {
		return rf(ctx, siteID)
	}

	var r0 *model.SiteAIInsight
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*model.SiteAIInsight)
	}
	return r0, ret.Error(1)
}

// CreateMetrics provides a mock function with given fields: ctx, metrics
func (_m *ISiteAIInsightRepository) CreateMetrics(ctx context.Context, metrics []*model.SiteAIMetric) error {
	ret := _m.Called(ctx, metrics)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.SiteAIMetric) error); ok {
		r0 = rf(ctx, metrics)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateRisks provides a mock function with given fields: ctx, risks
func (_m *ISiteAIInsightRepository) CreateRisks(ctx context.Context, risks []*model.SiteAIRisk) error {
	ret := _m.Called(ctx, risks)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.SiteAIRisk) error); ok {
		r0 = rf(ctx, risks)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateOpportunities provides a mock function with given fields: ctx, opportunities
func (_m *ISiteAIInsightRepository) CreateOpportunities(ctx context.Context, opportunities []*model.SiteAIOpportunity) error {
	ret := _m.Called(ctx, opportunities)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.SiteAIOpportunity) error); ok {
		r0 = rf(ctx, opportunities)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateDataAssumptions provides a mock function with given fields: ctx, assumptions
func (_m *ISiteAIInsightRepository) CreateDataAssumptions(ctx context.Context, assumptions []*model.SiteAIDataAssumption) error {
	ret := _m.Called(ctx, assumptions)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.SiteAIDataAssumption) error); ok {
		r0 = rf(ctx, assumptions)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateContextWarnings provides a mock function with given fields: ctx, warnings
func (_m *ISiteAIInsightRepository) CreateContextWarnings(ctx context.Context, warnings []*model.SiteAIContextWarning) error {
	ret := _m.Called(ctx, warnings)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*model.SiteAIContextWarning) error); ok {
		r0 = rf(ctx, warnings)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetMetrics provides a mock function with given fields: ctx, siteAIID
func (_m *ISiteAIInsightRepository) GetMetrics(ctx context.Context, siteAIID int64) ([]*model.SiteAIMetric, error) {
	ret := _m.Called(ctx, siteAIID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]*model.SiteAIMetric, error)); ok {
		return rf(ctx, siteAIID)
	}

	var r0 []*model.SiteAIMetric
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAIMetric)
	}
	return r0, ret.Error(1)
}

// GetRisks provides a mock function with given fields: ctx, siteAIID
func (_m *ISiteAIInsightRepository) GetRisks(ctx context.Context, siteAIID int64) ([]*model.SiteAIRisk, error) {
	ret := _m.Called(ctx, siteAIID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]*model.SiteAIRisk, error)); ok {
		return rf(ctx, siteAIID)
	}

	var r0 []*model.SiteAIRisk
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAIRisk)
	}
	return r0, ret.Error(1)
}

// GetOpportunities provides a mock function with given fields: ctx, siteAIID
func (_m *ISiteAIInsightRepository) GetOpportunities(ctx context.Context, siteAIID int64) ([]*model.SiteAIOpportunity, error) {
	ret := _m.Called(ctx, siteAIID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]*model.SiteAIOpportunity, error)); ok {
		return rf(ctx, siteAIID)
	}

	var r0 []*model.SiteAIOpportunity
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAIOpportunity)
	}
	return r0, ret.Error(1)
}

// GetDataAssumptions provides a mock function with given fields: ctx, siteAIID
func (_m *ISiteAIInsightRepository) GetDataAssumptions(ctx context.Context, siteAIID int64) ([]*model.SiteAIDataAssumption, error) {
	ret := _m.Called(ctx, siteAIID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]*model.SiteAIDataAssumption, error)); ok {
		return rf(ctx, siteAIID)
	}

	var r0 []*model.SiteAIDataAssumption
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAIDataAssumption)
	}
	return r0, ret.Error(1)
}

// GetContextWarnings provides a mock function with given fields: ctx, siteAIID
func (_m *ISiteAIInsightRepository) GetContextWarnings(ctx context.Context, siteAIID int64) ([]*model.SiteAIContextWarning, error) {
	ret := _m.Called(ctx, siteAIID)

	if rf, ok := ret.Get(0).(func(context.Context, int64) ([]*model.SiteAIContextWarning, error)); ok {
		return rf(ctx, siteAIID)
	}

	var r0 []*model.SiteAIContextWarning
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]*model.SiteAIContextWarning)
	}
	return r0, ret.Error(1)
}

// NewISiteAIInsightRepository creates a new instance of ISiteAIInsightRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISiteAIInsightRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ISiteAIInsightRepository {
	m := &ISiteAIInsightRepository{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
