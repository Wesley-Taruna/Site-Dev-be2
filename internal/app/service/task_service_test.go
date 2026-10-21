package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"
)

func testOptions() opt.Options {
	lgr := logrus.New()
	lgr.SetOutput(io.Discard)
	return opt.Options{
		Logger: log.NewLoggerWithClient("site-dev-backend-test", lgr),
		Trace:  noop.NewTracerProvider().Tracer("test"),
	}
}

type taskServiceMocks struct {
	task    *mocks.ITaskRepository
	history *mocks.ITaskHistoryRepository
	site    *mocks.ISiteRepository
	user    *mocks.IUserRepository
}

func newTestTaskService(t *testing.T) (*TaskService, taskServiceMocks) {
	m := taskServiceMocks{
		task:    mocks.NewITaskRepository(t),
		history: mocks.NewITaskHistoryRepository(t),
		site:    mocks.NewISiteRepository(t),
		user:    mocks.NewIUserRepository(t),
	}
	s := &TaskService{
		opt:                   testOptions(),
		RepositoryTask:        m.task,
		RepositoryTaskHistory: m.history,
		RepositorySite:        m.site,
		RepositoryUser:        m.user,
	}
	return s, m
}

var ctxAny = mock.Anything

func TestTaskServiceGet_StaffSeesOwnTasksWithSiteNames(t *testing.T) {
	s, m := newTestTaskService(t)
	created := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, RoleID: constants.RoleUser}, nil)
	m.task.On("Get", ctxAny, 20, 20, "CreatedAt ASC", []string(nil), constants.TaskStatusDraft, int64(1001)).
		Return([]*model.Task{
			{TaskID: "TS1", SiteID: "ST1", TaskStatus: constants.TaskStatusDraft, CreatedBy: 1001, CreatedAt: created},
			{TaskID: "TS2", SiteID: "ST9", TaskStatus: constants.TaskStatusDraft, CreatedBy: 1001, CreatedAt: created},
		}, int64(22), nil)
	m.site.On("GetByIDs", ctxAny, []string{"ST1", "ST9"}).
		Return([]*model.Site{{SiteID: "ST1", SiteName1: "Toko A", SiteAddress1: "Jl. Satu", SiteAddress3: "RT 1"}}, nil)

	res, total, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 2, Status: constants.TaskStatusDraft}, 1001)

	assert.NoError(t, err)
	assert.Equal(t, int64(22), total)
	assert.Equal(t, []dto.TaskResponseGet{
		{TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko A", SiteAddress1: "Jl. Satu", SiteAddress3: "RT 1", TaskStatus: "DRAFT", CreatedBy: 1001, CreatedAt: created},
		{TaskID: "TS2", SiteID: "ST9", TaskStatus: "DRAFT", CreatedBy: 1001, CreatedAt: created},
	}, res)
}

func TestTaskServiceGet_ApproverSeesAllTasksSorted(t *testing.T) {
	s, m := newTestTaskService(t)

	m.user.On("GetByID", ctxAny, int64(2002)).Return(&model.User{UserID: 2002, RoleID: constants.RoleDeptHead}, nil)
	m.task.On("Get", ctxAny, 0, 20, "TaskStatus DESC", []string{"ST5"}, "", int64(0)).
		Return([]*model.Task{}, int64(0), nil)
	m.site.On("GetByIDs", ctxAny, []string{}).Return(nil, nil)

	res, total, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 1, SiteID: "ST5", SortBy: "TaskStatus", SortDir: "-1"}, 2002)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.NotNil(t, res)
	assert.Len(t, res, 0)
}

func TestTaskServiceGet_SearchReplacesSiteFilter(t *testing.T) {
	s, m := newTestTaskService(t)

	m.user.On("GetByID", ctxAny, int64(2002)).Return(&model.User{UserID: 2002, RoleID: constants.RoleDivHead}, nil)
	m.site.On("GetIDsBySearch", ctxAny, "kemang").Return([]string{"ST7", "ST8"}, nil)
	m.task.On("Get", ctxAny, 0, 20, "CreatedAt ASC", []string{"ST7", "ST8"}, "", int64(0)).
		Return([]*model.Task{}, int64(0), nil)
	m.site.On("GetByIDs", ctxAny, []string{}).Return(nil, nil)

	_, _, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 1, SiteID: "ST5", Search: "kemang", SortBy: "SiteID"}, 2002)

	assert.NoError(t, err)
}

func TestTaskServiceGet_SearchWithoutHitsReturnsEmptyList(t *testing.T) {
	s, m := newTestTaskService(t)

	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, RoleID: constants.RoleUser}, nil)
	m.site.On("GetIDsBySearch", ctxAny, "zzz").Return([]string{}, nil)

	res, total, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 1, Search: "zzz"}, 1001)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, []dto.TaskResponseGet{}, res)
	m.task.AssertNotCalled(t, "Get")
}

func TestTaskServiceGet_UnknownUserIs401(t *testing.T) {
	s, m := newTestTaskService(t)

	m.user.On("GetByID", ctxAny, int64(9)).Return(nil, gorm.ErrRecordNotFound)

	_, _, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 1}, 9)

	var he *echo.HTTPError
	assert.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusUnauthorized, he.Code)
	assert.Equal(t, "user not found", he.Message)
}

func TestTaskServiceGet_UserLookupFailureIs500(t *testing.T) {
	s, m := newTestTaskService(t)

	m.user.On("GetByID", ctxAny, int64(9)).Return(nil, errors.New("connection reset"))

	_, _, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 1}, 9)

	var he *echo.HTTPError
	assert.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusInternalServerError, he.Code)
	assert.Equal(t, "failed to verify user", he.Message)
}

func TestTaskServiceGet_RepositoryErrorIsReturned(t *testing.T) {
	s, m := newTestTaskService(t)
	dbErr := errors.New("db down")

	m.user.On("GetByID", ctxAny, int64(1001)).Return(&model.User{UserID: 1001, RoleID: constants.RoleUser}, nil)
	m.task.On("Get", ctxAny, 0, 20, "CreatedAt ASC", []string(nil), "", int64(1001)).Return(nil, int64(0), dbErr)

	_, _, err := s.Get(context.Background(), dto.TaskRequestGet{Page: 1}, 1001)

	assert.ErrorIs(t, err, dbErr)
}

func TestTaskServiceGetHistory(t *testing.T) {
	s, m := newTestTaskService(t)
	at := time.Date(2026, 10, 2, 10, 30, 0, 0, time.UTC)

	m.task.On("GetBySiteID", ctxAny, "ST1").Return(&model.Task{TaskID: "TS1", SiteID: "ST1"}, nil)
	m.history.On("GetByTaskID", ctxAny, "TS1").Return([]*model.TaskHistory{
		{ID: 1, TaskID: "TS1", Action: constants.TaskActionCreate, ActorID: 1001, Notes: "Site dibuat", CreatedAt: at},
		{ID: 2, TaskID: "TS1", Action: constants.TaskActionSubmit, ActorID: 1001, CreatedAt: at.Add(time.Hour)},
	}, nil)

	res, err := s.GetHistory(context.Background(), "ST1")

	assert.NoError(t, err)
	assert.Equal(t, []dto.TaskHistoryResponseGet{
		{ID: 1, Action: "CREATE", ActorID: 1001, Notes: "Site dibuat", CreatedAt: at},
		{ID: 2, Action: "SUBMIT", ActorID: 1001, CreatedAt: at.Add(time.Hour)},
	}, res)
}

func TestTaskServiceGetHistory_NoHistoryIsEmptyList(t *testing.T) {
	s, m := newTestTaskService(t)

	m.task.On("GetBySiteID", ctxAny, "ST1").Return(&model.Task{TaskID: "TS1", SiteID: "ST1"}, nil)
	m.history.On("GetByTaskID", ctxAny, "TS1").Return(nil, nil)

	res, err := s.GetHistory(context.Background(), "ST1")

	assert.NoError(t, err)
	assert.Equal(t, []dto.TaskHistoryResponseGet{}, res)
}

func TestTaskServiceGetHistory_UnknownSiteReturnsNotFound(t *testing.T) {
	s, m := newTestTaskService(t)

	m.task.On("GetBySiteID", ctxAny, "ST404").Return(nil, gorm.ErrRecordNotFound)

	_, err := s.GetHistory(context.Background(), "ST404")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestTaskOrderBy(t *testing.T) {
	cases := []struct{ by, dir, want string }{
		{"", "", "CreatedAt ASC"},
		{"TaskStatus", "desc", "TaskStatus DESC"},
		{"TaskStatus", "DESC", "TaskStatus DESC"},
		{"CreatedAt", "-1", "CreatedAt DESC"},
		{"CreatedAt", "1", "CreatedAt ASC"},
		{"SiteID; DROP TABLE Tasks", "desc", "CreatedAt DESC"},
		{"TaskStatus", "sideways", "TaskStatus ASC"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, taskOrderBy(c.by, c.dir), c.by+"/"+c.dir)
	}
}
