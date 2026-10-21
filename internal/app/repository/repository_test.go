package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"site-dev-backend/internal/app/model"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sqlRecorder is a gorm logger that keeps every SQL statement gorm runs
// (with values filled in). The database is sqlfake (in memory), so these
// tests check the SQL Server queries without a live DB.
type sqlRecorder struct {
	mu   sync.Mutex
	sqls []string
}

func (l *sqlRecorder) LogMode(logger.LogLevel) logger.Interface      { return l }
func (l *sqlRecorder) Info(context.Context, string, ...interface{})  {}
func (l *sqlRecorder) Warn(context.Context, string, ...interface{})  {}
func (l *sqlRecorder) Error(context.Context, string, ...interface{}) {}
func (l *sqlRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.mu.Lock()
	l.sqls = append(l.sqls, sql)
	l.mu.Unlock()
}

func fakeDBOptions(t *testing.T, results ...sqlfake.Result) (opt.Options, *sqlRecorder) {
	rec := &sqlRecorder{}
	conn, err := sqlfake.Open(&sqlfake.DB{Results: results})
	require.NoError(t, err)
	db, err := gorm.Open(sqlserver.New(sqlserver.Config{Conn: conn}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               rec,
	})
	require.NoError(t, err)

	lgr := logrus.New()
	lgr.SetOutput(io.Discard)
	return opt.Options{
		Database: opt.Database{Read: db, Write: db},
		Logger:   log.NewLoggerWithClient("site-dev-backend-test", lgr),
		Trace:    noop.NewTracerProvider().Tracer("test"),
	}, rec
}

func TestTaskRepositoryGet_AllFilters(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	_, _, err := r.Get(context.Background(), 20, 20, "CreatedAt ASC", []string{"ST1", "ST2"}, "DRAFT", 1001)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"SELECT count(*) FROM \"Tasks\" WHERE SiteID IN ('ST1','ST2') AND TaskStatus = 'DRAFT' AND CreatedBy = 1001",
		"SELECT * FROM \"Tasks\" WHERE SiteID IN ('ST1','ST2') AND TaskStatus = 'DRAFT' AND CreatedBy = 1001 ORDER BY CreatedAt ASC OFFSET 20 ROWS FETCH NEXT 20 ROWS ONLY",
	}, rec.sqls)
}

func TestTaskRepositoryGet_NoFiltersFirstPage(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	_, _, err := r.Get(context.Background(), 0, 20, "TaskStatus DESC", nil, "", 0)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"SELECT count(*) FROM \"Tasks\"",
		"SELECT * FROM \"Tasks\" ORDER BY TaskStatus DESC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY",
	}, rec.sqls)
}

func TestTaskRepositoryGetBySiteID(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	_, _ = r.GetBySiteID(context.Background(), "ST1")

	assert.Equal(t, []string{
		"SELECT * FROM \"Tasks\" WHERE SiteID = 'ST1' ORDER BY \"Tasks\".\"TaskID\" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY",
	}, rec.sqls)
}

func TestTaskHistoryRepositoryGetByTaskID(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskHistoryRepository{opt: o}

	_, err := r.GetByTaskID(context.Background(), "TS1")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"SELECT * FROM \"TaskHistories\" WHERE TaskID = 'TS1' ORDER BY CreatedAt ASC",
	}, rec.sqls)
}

func TestSiteRepositoryGetByIDs(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &SiteRepository{opt: o}

	sites, err := r.GetByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, sites)
	assert.Empty(t, rec.sqls, "no query for an empty ID list")

	_, err = r.GetByIDs(context.Background(), []string{"ST1", "ST2"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"SELECT * FROM \"Sites\" WHERE SiteID IN ('ST1','ST2')",
	}, rec.sqls)
}

func TestSiteRepositoryGetIDsBySearch(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &SiteRepository{opt: o}

	_, err := r.GetIDsBySearch(context.Background(), "kemang")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"SELECT \"SiteID\" FROM \"Sites\" WHERE SiteID LIKE '%kemang%' OR SiteName1 LIKE '%kemang%' OR SiteName2 LIKE '%kemang%' OR SiteAddress1 LIKE '%kemang%' OR SiteAddress2 LIKE '%kemang%' OR SiteAddress3 LIKE '%kemang%'",
	}, rec.sqls)
}

func TestUserRepositoryGetByID(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &UserRepository{opt: o}

	_, _ = r.GetByID(context.Background(), 1001)

	assert.Equal(t, []string{
		"SELECT * FROM \"Users\" WHERE UserID = 1001 ORDER BY \"Users\".\"UserID\" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY",
	}, rec.sqls)
}

func TestTaskRepositoryGetBySiteID_MapsColumns(t *testing.T) {
	created := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	o, _ := fakeDBOptions(t, sqlfake.Result{
		Match:   `FROM "Tasks"`,
		Columns: []string{"TaskID", "SiteID", "TaskStatus", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt"},
		Rows:    [][]driver.Value{{"TS1", "ST1", "WAITING_DEPT_HEAD", int64(1001), int64(2002), created, created.Add(time.Hour)}},
	})
	r := &TaskRepository{opt: o}

	task, err := r.GetBySiteID(context.Background(), "ST1")

	require.NoError(t, err)
	assert.Equal(t, &model.Task{
		TaskID: "TS1", SiteID: "ST1", TaskStatus: "WAITING_DEPT_HEAD",
		CreatedBy: 1001, UpdatedBy: 2002, CreatedAt: created, UpdatedAt: created.Add(time.Hour),
	}, task)
}

func TestTaskRepositoryGetBySiteID_NotFound(t *testing.T) {
	o, _ := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	task, err := r.GetBySiteID(context.Background(), "ST404")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Nil(t, task)
}

func TestTaskRepositoryGet_ErrorIsReturned(t *testing.T) {
	dbErr := errors.New("db down")
	o, _ := fakeDBOptions(t, sqlfake.Result{Match: "count(*)", Err: dbErr})
	r := &TaskRepository{opt: o}

	_, _, err := r.Get(context.Background(), 0, 20, "CreatedAt ASC", nil, "", 0)

	assert.ErrorIs(t, err, dbErr)
}
