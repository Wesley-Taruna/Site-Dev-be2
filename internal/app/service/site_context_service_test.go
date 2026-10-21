package service

import (
	"context"
	"testing"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSiteContextServiceUpdate_RollsBackOnHistoryFailure(t *testing.T) {
	users, sites, tasks, contexts, histories, tx := mocks.NewIUserRepository(t), mocks.NewISiteRepository(t), mocks.NewITaskRepository(t),
		mocks.NewISiteContextRepository(t), mocks.NewITaskHistoryRepository(t), mocks.NewITransactionRepository(t)
	s := &SiteContextService{opt: testOptions(), RepositoryUser: users, RepositorySite: sites, RepositoryTask: tasks,
		RepositorySiteContext: contexts, RepositoryTaskHistory: histories, RepositoryTransaction: tx}
	users.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, Username: "andi"}, nil)
	sites.On("GetByID", ctxAny, "ST1").Return(&model.Site{SiteID: "ST1", CreatedBy: 1001}, nil)
	tasks.On("GetBySiteID", ctxAny, "ST1").Return(&model.Task{TaskID: "TS1", TaskStatus: "DRAFT"}, nil)
	tx.On("Do", ctxAny, mock.Anything).Return(func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) })
	contexts.On("Upsert", ctxAny, mock.Anything).Return(nil)
	contexts.On("ReplaceCompetitors", ctxAny, "ST1", []*model.SiteCompetitor(nil)).Return(nil)
	contexts.On("ReplaceCustomNotes", ctxAny, "ST1", []*model.SiteCustomNote(nil)).Return(nil)
	histories.On("Create", ctxAny, mock.MatchedBy(func(h *model.TaskHistory) bool {
		return h.Notes == "Konteks on-site diperbarui oleh andi" && h.Action == "UPDATE" && h.TaskID == "TS1"
	})).Return(assert.AnError)

	res, err := s.Update(context.Background(), "ST1", dto.SiteContextRequestUpsert{}, 1001)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, dto.SiteContextResponseGet{}, res)
}
