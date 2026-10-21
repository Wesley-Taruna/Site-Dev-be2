package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func expectDetailOfST1(m taskServiceMocks, creator int64) {
	m.task.On("GetBySiteID", ctxAny, "ST1").Return(&model.Task{TaskID: "TS1", SiteID: "ST1", TaskStatus: "DRAFT"}, nil)
	m.site.On("GetByID", ctxAny, "ST1").Return(&model.Site{SiteID: "ST1", CreatedBy: creator}, nil)
	m.context.On("GetBySiteID", ctxAny, "ST1").Return(nil, gorm.ErrRecordNotFound)
	m.history.On("GetByTaskID", ctxAny, "TS1").Return(nil, nil)
	m.approval.On("GetHeaderByTaskID", ctxAny, "TS1").Return(nil, gorm.ErrRecordNotFound)
	m.site.On("GetAttachmentsBySiteID", ctxAny, "ST1").Return(nil, nil)
}

func TestTaskServiceGetDetail_OwnerSeesDraftWithoutContextOrApproval(t *testing.T) {
	s, m := newTestTaskService(t)
	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, Username: "andi", RoleID: constants.RoleUser}, nil)
	expectDetailOfST1(m, 1001)

	res, err := s.GetDetail(context.Background(), "ST1", 1001)

	require.NoError(t, err)
	assert.Equal(t, "andi", res.Site.CreatedByName)
	assert.Nil(t, res.Context)
	assert.Nil(t, res.Task.Approval)
	assert.Nil(t, res.Attachment)
	m.context.AssertNotCalled(t, "GetCompetitorsBySiteID")
	m.approval.AssertNotCalled(t, "GetDetailsByHeaderID")
}

func TestTaskServiceGetDetail_StaffCannotOpenOtherSites(t *testing.T) {
	s, m := newTestTaskService(t)
	m.user.On("GetByID", ctxAny, int64(5005)).Return(&model.User{UserID: 5005, RoleID: constants.RoleUser}, nil)
	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001}, nil)
	expectDetailOfST1(m, 1001)

	res, err := s.GetDetail(context.Background(), "ST1", 5005)

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusForbidden, he.Code)
	assert.Equal(t, "Anda tidak punya akses ke task TS1", he.Message)
	assert.Equal(t, dto.TaskDetailResponseGet{}, res)
}

func TestTaskServiceGetDetail_ApproverOpensAnySite(t *testing.T) {
	s, m := newTestTaskService(t)
	m.user.On("GetByID", ctxAny, int64(3001)).Return(&model.User{UserID: 3001, RoleID: constants.RoleDeptHead}, nil)
	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001}, nil)
	expectDetailOfST1(m, 1001)

	_, err := s.GetDetail(context.Background(), "ST1", 3001)

	assert.NoError(t, err)
}

func TestTaskServiceGetDetail_ContextLookupFailureIsReturned(t *testing.T) {
	s, m := newTestTaskService(t)
	dbErr := errors.New("db down")
	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, RoleID: constants.RoleAdmin}, nil)
	m.task.On("GetByID", ctxAny, "TS1").Return(&model.Task{TaskID: "TS1", SiteID: "ST1"}, nil)
	m.site.On("GetByID", ctxAny, "ST1").Return(&model.Site{SiteID: "ST1", CreatedBy: 1001}, nil)
	m.context.On("GetBySiteID", ctxAny, "ST1").Return(nil, dbErr)

	_, err := s.GetDetail(context.Background(), "TS1", 1001)

	assert.ErrorIs(t, err, dbErr)
}

func TestTaskServiceCountByStatus_StaffCountsOwnTasks(t *testing.T) {
	s, m := newTestTaskService(t)
	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, RoleID: constants.RoleUser}, nil)
	m.task.On("CountByStatus", ctxAny, int64(1001)).Return([]model.TaskStatusCount{{TaskStatus: "DRAFT", Count: 2}}, nil)

	res, err := s.CountByStatus(context.Background(), 1001)

	require.NoError(t, err)
	assert.Equal(t, []dto.TaskStatusCountResponseGet{{TaskStatus: "DRAFT", Count: 2}}, res)
}

func TestTaskServiceGetDashboardStatusCount_EmptyIsArray(t *testing.T) {
	s, m := newTestTaskService(t)
	m.task.On("CountByStatus", ctxAny, int64(0)).Return(nil, nil)

	res, err := s.GetDashboardStatusCount(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []dto.TaskStatusCountResponseGet{}, res)
}

func TestTaskServiceGetSiteList_SearchWithoutHits(t *testing.T) {
	s, m := newTestTaskService(t)
	m.site.On("GetIDsBySearch", ctxAny, "zzz").Return(nil, nil)

	res, total, err := s.GetSiteList(context.Background(), dto.ListRequestGet{Page: 1, Search: "zzz"})

	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, []dto.TaskResponseGet{}, res)
	m.task.AssertNotCalled(t, "GetByStatuses")
}

func TestTaskServiceGetSiteList_StatusesAndPaging(t *testing.T) {
	s, m := newTestTaskService(t)
	m.task.On("GetByStatuses", ctxAny, 20, 20, "CreatedAt ASC", constants.SiteListRequestStatuses, []string(nil)).Return(nil, int64(0), nil)
	m.site.On("GetByIDs", ctxAny, []string{}).Return(nil, nil)

	_, _, err := s.GetSiteList(context.Background(), dto.ListRequestGet{Page: 2})

	assert.NoError(t, err)
	assert.NotContains(t, constants.SiteListRequestStatuses, constants.TaskStatusApproved)
}

func TestTaskServiceGetPendingApprovals_ErrorMidListReturnsNothing(t *testing.T) {
	s, m := newTestTaskService(t)
	dbErr := errors.New("db down")
	m.approval.On("GetPendingByApprover", ctxAny, int64(3001), "", "ApprovalDET.CreatedAt ASC", 20, 0).
		Return([]*model.ApprovalDetail{{ID: 1, ApprovalHeaderID: 7}}, int64(1), nil)
	m.approval.On("GetHeaderByID", ctxAny, uint(7)).Return(nil, dbErr)

	res, total, err := s.GetPendingApprovals(context.Background(), dto.ListRequestGet{Page: 1}, 3001)

	assert.ErrorIs(t, err, dbErr)
	assert.Nil(t, res)
	assert.Zero(t, total)
}

func TestTaskServiceGetApprovalHistory_CreatorNameIsBestEffort(t *testing.T) {
	s, m := newTestTaskService(t)
	at := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	m.approval.On("GetActedByApprover", ctxAny, int64(3001), "kem", "ApprovalDET.Level DESC", 20, 0).
		Return([]*model.ApprovalDetail{{ID: 1, ApprovalHeaderID: 7, Level: 1, Status: "APPROVED", ActedAt: &at}}, int64(1), nil)
	m.approval.On("GetHeaderByID", ctxAny, uint(7)).Return(&model.ApprovalHeader{ID: 7, TaskID: "TS1"}, nil)
	m.task.On("GetByID", ctxAny, "TS1").Return(&model.Task{TaskID: "TS1", SiteID: "ST1", TaskStatus: "COMPLETED"}, nil)
	m.site.On("GetByID", ctxAny, "ST1").Return(&model.Site{SiteID: "ST1", CreatedBy: 1001}, nil)
	m.user.On("GetByID", ctxAny, int64(1001)).Return(nil, gorm.ErrRecordNotFound)

	res, total, err := s.GetApprovalHistory(context.Background(), dto.ListRequestGet{Page: 1, Search: "kem", SortBy: "Level", SortDir: "desc"}, 3001)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, res, 1)
	assert.Equal(t, "", res[0].CreatedByName)
	assert.Equal(t, "COMPLETED", res[0].TaskStatus)
}

func TestTaskServiceGetDashboardHistory_NoRowsNoLookups(t *testing.T) {
	s, m := newTestTaskService(t)
	m.history.On("Get", ctxAny, 0, 20, "CreatedAt ASC").Return(nil, int64(0), nil)

	res, total, err := s.GetDashboardHistory(context.Background(), dto.ListRequestGet{})

	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Equal(t, []dto.DashboardTaskHistoryResponseGet{}, res)
	m.task.AssertNotCalled(t, "GetByIDs")
}
