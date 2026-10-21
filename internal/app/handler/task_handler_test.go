package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"site-dev-backend/global"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/middleware"
	"site-dev-backend/internal/app/mocks"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/fami-app-v2/familib/familabs"
	jwtx "gitlab.com/fami-app-v2/familib/familabs/jwt"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"go.opentelemetry.io/otel/trace/noop"
)

// test-only signing key, not used anywhere else
const testJwtKey = "unit-test-signing-key"

func testHandlerOptions() global.HandlerOptions {
	lgr := logrus.New()
	lgr.SetOutput(io.Discard)
	o := opt.Options{
		Logger: log.NewLoggerWithClient("site-dev-backend-test", lgr),
		Trace:  noop.NewTracerProvider().Tracer("test"),
	}
	o.Config.Jwt.Key = testJwtKey
	return global.HandlerOptions{Common: o}
}

// newTestEcho mirrors router.Router(): familabs context, binder and error
// handler; routes as in TaskHandler.URLMapping but with a mocked service.
func newTestEcho(t *testing.T, svc *mocks.ITaskService) *echo.Echo {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(&familabs.Context{Context: c, ResponseFormat: familabs.NewResponse()})
		}
	})
	e.Binder = &familabs.Binder{}
	e.HTTPErrorHandler = familabs.HTTPErrorHandler

	o := testHandlerOptions()
	h := TaskHandler{Option: o, ServicesTask: svc}
	mw := &middleware.Middleware{Option: o.Common}

	g := e.Group("v1/").Group("task")
	g.GET("", h.Index, mw.Authorized())
	g.GET("/:id/history", h.History, mw.Authorized())
	return e
}

func testToken(t *testing.T, userID int64, expiresAt time.Time) string {
	token, err := jwtx.NewJWT([]byte(testJwtKey)).Create(&jwtx.UserClaim{
		UserID:    userID,
		ExpiresAt: expiresAt.Unix(),
	})
	require.NoError(t, err)
	return token
}

func doGet(e *echo.Echo, path, token string) (int, map[string]any) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestTaskHandlerIndex(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)
	created := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	svc.On("Get", mock.Anything, dto.TaskRequestGet{
		Page: 2, SiteID: "ST1", Status: "DRAFT", Search: "kemang", SortBy: "TaskStatus", SortDir: "desc",
	}, int64(1001)).Return([]dto.TaskResponseGet{
		{TaskID: "TS1", SiteID: "ST1", TaskStatus: "DRAFT", CreatedBy: 1001, CreatedAt: created, UpdatedAt: created},
	}, int64(21), nil)

	code, body := doGet(e, "/v1/task?page=2&site_id=ST1&status=DRAFT&q=kemang&sort_by=TaskStatus&sort_dir=desc",
		testToken(t, 1001, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, float64(200), body["code"])
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, float64(21), body["total"])
	assert.Equal(t, float64(1), body["page"])      // per_page not sent: familabs reports page 1
	assert.Equal(t, float64(21), body["per_page"]) // and per_page = total
	rows := body["data"].([]any)
	require.Len(t, rows, 1)
	assert.Equal(t, "TS1", rows[0].(map[string]any)["task_id"])
}

func TestTaskHandlerIndex_EmptyListIsArray(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)

	svc.On("Get", mock.Anything, dto.TaskRequestGet{Page: 1}, int64(1001)).Return([]dto.TaskResponseGet{}, int64(0), nil)

	code, body := doGet(e, "/v1/task", testToken(t, 1001, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, []any{}, body["data"])
	assert.NotContains(t, body, "total")
}

func TestTaskHandlerIndex_InvalidPageIs422(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)

	code, body := doGet(e, "/v1/task?page=0", testToken(t, 1001, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Equal(t, "failure", body["status"])
	svc.AssertNotCalled(t, "Get")
}

func TestTaskHandlerIndex_ServiceHTTPErrorKeepsCode(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)

	svc.On("Get", mock.Anything, dto.TaskRequestGet{Page: 1}, int64(9)).
		Return(nil, int64(0), echo.NewHTTPError(http.StatusUnauthorized, "user not found"))

	code, body := doGet(e, "/v1/task", testToken(t, 9, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Equal(t, map[string]any{"error": "user not found"}, body["errors"])
}

func TestTaskHandlerHistory(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)
	at := time.Date(2026, 10, 2, 10, 30, 0, 0, time.UTC)

	svc.On("GetHistory", mock.Anything, "ST1").Return([]dto.TaskHistoryResponseGet{
		{ID: 1, Action: "CREATE", ActorID: 1001, Notes: "Site dibuat", CreatedAt: at},
	}, nil)

	code, body := doGet(e, "/v1/task/ST1/history", testToken(t, 1001, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, []any{map[string]any{
		"id": float64(1), "action": "CREATE", "actor_id": float64(1001), "notes": "Site dibuat", "created_at": "2026-10-02T10:30:00Z",
	}}, body["data"])
}

func TestTaskHandlerHistory_ErrorIs500(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)

	svc.On("GetHistory", mock.Anything, "ST404").Return(nil, errors.New("record not found"))

	code, body := doGet(e, "/v1/task/ST404/history", testToken(t, 1001, time.Now().Add(time.Hour)))

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, map[string]any{"error": "record not found"}, body["errors"])
}

func TestTaskHandler_Auth(t *testing.T) {
	svc := mocks.NewITaskService(t)
	e := newTestEcho(t, svc)

	// no token: familabs Authorization answers 403
	code, _ := doGet(e, "/v1/task", "")
	assert.Equal(t, http.StatusForbidden, code)

	// wrong signature
	bad, err := jwtx.NewJWT([]byte("other-key")).Create(&jwtx.UserClaim{UserID: 1, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)
	code, _ = doGet(e, "/v1/task", bad)
	assert.Equal(t, http.StatusUnauthorized, code)

	// expired (familabs reads the "expire_at" claim)
	code, _ = doGet(e, "/v1/task/ST1/history", testToken(t, 1, time.Now().Add(-time.Minute)))
	assert.Equal(t, http.StatusUnauthorized, code)

	svc.AssertNotCalled(t, "Get")
	svc.AssertNotCalled(t, "GetHistory")
}
