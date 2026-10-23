package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestActivityService_MarkDivHeadReturns(t *testing.T) {
	at := time.Date(2026, 10, 4, 10, 30, 0, 0, time.UTC)
	round2 := 2
	history := mocks.NewITaskHistoryRepository(t)
	history.On("GetByTaskIDsAndActor", mock.Anything, []string{"TS1", "TS2"}, int64(3001), constants.TaskActionApprove).
		Return([]*model.TaskHistory{
			{TaskID: "TS1", CreatedAt: at.Add(-time.Hour)},                 // round 1 (NULL), before: notice
			{TaskID: "TS2", CreatedAt: at.Add(-time.Hour), Round: &round2}, // other round: no notice
		}, nil)
	s := &ActivityService{RepositoryTaskHistory: history}
	res := []dto.ActivityResponseGet{
		{TaskID: "TS1", Action: constants.TaskActionRevision, ActorRole: constants.RoleDivHead, Round: 1, CreatedAt: at},
		{TaskID: "TS2", Action: constants.TaskActionRevision, ActorRole: constants.RoleDivHead, Round: 1, CreatedAt: at},
		{TaskID: "TS1", Action: constants.TaskActionRevision, ActorRole: constants.RoleDeptHead, Round: 1, CreatedAt: at},
	}

	assert.NoError(t, s.markDivHeadReturns(context.Background(), res, 3001))
	assert.Equal(t, []string{constants.ApproverNewsDivHeadReturned, "", ""}, []string{res[0].Notice, res[1].Notice, res[2].Notice})

	// nothing to check: no query
	assert.NoError(t, s.markDivHeadReturns(context.Background(), res[2:], 3001))

	failing := mocks.NewITaskHistoryRepository(t)
	failing.On("GetByTaskIDsAndActor", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
	assert.Error(t, (&ActivityService{RepositoryTaskHistory: failing}).markDivHeadReturns(context.Background(), res[:1], 3001))
}
