// Written in the layout mockery (v2) generates, because mockery is not
// installed on this machine. Regenerate with mockery once it is (see
// README "Testing"); the type and constructor names stay the same.

package mocks

import (
	"context"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/mock"
)

// IGeminiClient is a mock type for the client.IGeminiClient type
type IGeminiClient struct {
	mock.Mock
}

// GenerateInsight provides a mock function with given fields: ctx, in
func (_m *IGeminiClient) GenerateInsight(ctx context.Context, in client.
	GeminiInput) (dto.AIInsightResponseGet, string, error) {
	ret := _m.Called(ctx, in)

	if rf, ok := ret.Get(0).(func(context.Context, client.
		GeminiInput) (dto.AIInsightResponseGet, string, error)); ok {
		return rf(ctx, in)
	}

	return ret.Get(0).(dto.AIInsightResponseGet), ret.Get(1).(string), ret.Error(2)
}

// NewIGeminiClient creates a new instance of IGeminiClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIGeminiClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *IGeminiClient {
	m := &IGeminiClient{}
	m.Mock.Test(t)

	t.Cleanup(func() { m.AssertExpectations(t) })

	return m
}
