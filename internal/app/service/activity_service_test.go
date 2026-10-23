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

func TestActivityNoteOf(t *testing.T) {
	cases := []struct {
		name  string
		notes string
		want  string
	}{
		{"return to staff", "Site diminta revisi oleh dewi (Div Head) dengan catatan (Foto kurang)", "Foto kurang"},
		{"hold", "Site ditahan (Hold Status) oleh citra (Dept Head) dengan alasan (Tunggu zonasi (RT 05))", "Tunggu zonasi (RT 05)"},
		{"continue without note", "Hold Status dilepas oleh citra (Dept Head)", ""},
		{"approve without note", "Site di approve oleh citra (Dept Head)", ""},
		{"empty reason", "Site diminta revisi oleh dewi (Div Head) dengan catatan ()", ""},
		{"send with note", "Approval dikirim ke Dept Head, status PENDING (catatan: Dekat stasiun)", "Dekat stasiun"},
		{"send, automatic text", "Approval dikirim ke Dept Head, status PENDING (catatan: Site diajukan oleh andi untuk proses approval)", ""},
		{"send without note", "Approval dikirim ke Dept Head, status PENDING", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, activityNoteOf(c.notes))
		})
	}
}

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
