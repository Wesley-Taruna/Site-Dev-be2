package service

import (
	"context"
	"testing"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParseDistanceKm(t *testing.T) {
	for in, want := range map[string]float64{"500m": 0.5, "2km": 2, "2": 2, " 1.5 KM ": 1.5, "dekat": 0, "": 0, "120 m": 0.12} {
		assert.InDelta(t, want, parseDistanceKm(in), 1e-9, in)
	}
}

func TestMetricRows_FixedOrder(t *testing.T) {
	rows := metricRows(7, dto.AIInsightMetrics{Economic: dto.AIInsightMetric{Label: "Eko", Factors: []string{"a", "b"}}})
	require.Len(t, rows, 5) // no vehicleTraffic: 5 rows as before
	for i, key := range constants.MetricKeyOrder[:5] {
		assert.Equal(t, key, rows[i].MetricKey)
		assert.Equal(t, int64(7), rows[i].SiteAIID)
	}
	assert.Equal(t, "a;b", rows[4].Factors)

	// November scope: the 6th metric is the 6th row
	rows = metricRows(7, dto.AIInsightMetrics{VehicleTraffic: &dto.AIInsightMetric{Label: "Lalu Lintas Kendaraan", Score: 72}})
	require.Len(t, rows, 6)
	assert.Equal(t, "vehicleTraffic", rows[5].MetricKey)
	assert.Equal(t, 72, rows[5].Score)
}

func TestAIInsightServicePersist_UsesSiteIDAndNoModel(t *testing.T) {
	insights, tx := mocks.NewISiteAIInsightRepository(t), mocks.NewITransactionRepository(t)
	s := &AIInsightService{opt: testOptions(), RepositorySiteAIInsight: insights, RepositoryTransaction: tx}
	tx.On("Do", ctxAny, mock.Anything).Return(func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) })
	insights.On("Replace", ctxAny, mock.MatchedBy(func(r *model.SiteAIInsight) bool {
		return r.SiteID == "ST5" && r.GeminiModel == "" && r.GeneratedBy == 1001
	})).Run(func(args mock.Arguments) { args.Get(1).(*model.SiteAIInsight).SiteAIID = 9 }).Return(nil)
	insights.On("CreateMetrics", ctxAny, mock.MatchedBy(func(m []*model.SiteAIMetric) bool { return len(m) == 5 && m[0].SiteAIID == 9 })).Return(nil)
	insights.On("CreateRisks", ctxAny, []*model.SiteAIRisk{}).Return(nil)
	insights.On("CreateOpportunities", ctxAny, []*model.SiteAIOpportunity{}).Return(nil)
	insights.On("CreateDataAssumptions", ctxAny, []*model.SiteAIDataAssumption{}).Return(nil)
	insights.On("CreateContextWarnings", ctxAny, []*model.SiteAIContextWarning{}).Return(nil)

	err := s.Persist(context.Background(), "ST5", dto.AIInsightResponseGet{SiteID: "OTHER"}, 1001)

	assert.NoError(t, err)
}
