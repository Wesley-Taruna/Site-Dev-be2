package handler

import (
	"net/http"
	"testing"
	"time"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskHandlerDetail(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)
	svc.On("GetDetail", mock.Anything, "ST1", int64(1001)).Return(dto.TaskDetailResponseGet{Task: dto.TaskDetailTaskSection{TaskID: "TS1"}}, nil)

	code, body := doGet(e, "/v1/task/ST1", testToken(t, 1001, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "TS1", body["data"].(map[string]any)["task"].(map[string]any)["task_id"])
}

func TestTaskHandlerDetail_Forbidden(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)
	svc.On("GetDetail", mock.Anything, "ST1", int64(5005)).Return(dto.TaskDetailResponseGet{}, echo.NewHTTPError(http.StatusForbidden, "Anda tidak punya akses ke task TS1"))

	code, body := doGet(e, "/v1/task/ST1", testToken(t, 5005, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusForbidden, code)
	assert.Equal(t, map[string]any{"error": "Anda tidak punya akses ke task TS1"}, body["errors"])
}

func TestTaskHandlerStaticRoutesWinOverID(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)
	token := testToken(t, 3001, time.Now().Add(time.Hour))
	svc.On("GetSiteList", mock.Anything, dto.ListRequestGet{Page: 2, SortBy: "TaskStatus", SortDir: "desc", Search: "kem"}).Return([]dto.TaskResponseGet{}, int64(0), nil)
	svc.On("GetPendingApprovals", mock.Anything, dto.ListRequestGet{Page: 1}, int64(3001)).Return([]dto.PendingApprovalResponseGet{}, int64(0), nil)
	svc.On("GetApprovalHistory", mock.Anything, dto.ListRequestGet{Page: 1}, int64(3001)).Return([]dto.ApprovalHistoryResponseGet{}, int64(0), nil)
	svc.On("CountByStatus", mock.Anything, int64(3001)).Return([]dto.TaskStatusCountResponseGet{}, nil)
	svc.On("GetAttachments", mock.Anything, "ST1").Return([]dto.TaskAttachmentResponseGet{}, nil)
	svc.On("GetDashboardHistory", mock.Anything, dto.ListRequestGet{Page: 1}).Return([]dto.DashboardTaskHistoryResponseGet{}, int64(0), nil)
	svc.On("GetDashboardStatusCount", mock.Anything).Return([]dto.TaskStatusCountResponseGet{}, nil)

	for _, path := range []string{
		"/v1/task/site-list?page=2&sort_by=TaskStatus&sort_dir=desc&q=kem",
		"/v1/task/pending-approvals",
		"/v1/task/approval-history",
		"/v1/task/status/count",
		"/v1/task/ST1/attachment",
		"/v1/dashboard/task/history",
		"/v1/dashboard/task/status/count",
	} {
		code, body := doGet(e, path, token)
		assert.Equal(t, http.StatusOK, code, path)
		assert.Equal(t, []any{}, body["data"], path)
	}
	svc.AssertNotCalled(t, "GetDetail")
}

func TestTaskHandlerReads_NeedToken(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)

	for _, path := range []string{"/v1/task/ST1", "/v1/task/status/count", "/v1/dashboard/task/history"} {
		code, _ := doGet(e, path, "")
		assert.Equal(t, http.StatusForbidden, code, path) // boilerplate Authorized(): no token = 403 (plan §4 diff 1)
	}
}
