package router

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/pkg/redisfake"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	jwtx "gitlab.com/fami-app-v2/familib/familabs/jwt"
	"gitlab.com/fami-app-v2/familib/familabs/log"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gitlab.com/fami-app-v2/familib/familabs/redisx"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// gormNow is what gorm writes into CreatedAt/UpdatedAt fields it fills.
var gormNow = time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)

// testEnv holds env.toml values the tests need (no real keys).
var testEnv = viper.New()

// stack builds the whole new service (Router, real middleware, handlers,
// services, repositories, gorm SQL Server dialect) on an in-memory database
// that answers with results.
func stack(t *testing.T, results ...sqlfake.Result) (*echo.Echo, *sqlLog, *sqlfake.DB) {
	fdb := &sqlfake.DB{Results: results}
	conn, err := sqlfake.Open(fdb)
	require.NoError(t, err)
	e, rec, _ := stackOn(t, conn)
	return e, rec, fdb
}

func stackOn(t *testing.T, conn *sql.DB) (*echo.Echo, *sqlLog, *opt.Options) {
	rec := &sqlLog{}
	db, err := gorm.Open(sqlserver.New(sqlserver.Config{Conn: conn}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               rec,
		NowFunc:              func() time.Time { return gormNow },
	})
	require.NoError(t, err)

	lgr := logrus.New()
	lgr.SetOutput(io.Discard)
	o := opt.Options{
		Database: opt.Database{Read: db, Write: db},
		Logger:   log.NewLoggerWithClient("site-dev-backend-test", lgr),
		Trace:    noop.NewTracerProvider().Tracer("test"),
		Env:      testEnv,
	}
	o.Config.App.Name = "site-dev-api"
	o.Config.App.Env = "test"
	o.Config.Jwt.Key = testJwtKey
	if redisServer != nil {
		o.Redisx = redisx.NewRedisx(redis.NewClient(&redis.Options{Addr: redisServer.Addr()}))
	}
	global.Setup = global.HandlerOptions{Common: o}

	e := Router()
	lgr.SetOutput(io.Discard) // Router() points the logger at stdout
	return e, rec, &global.Setup.Common
}

// redisServer, when set by withRedis, is given to the next stack.
var redisServer *redisfake.Server

// withRedis starts an in-memory Redis for the stacks built in this test.
func withRedis(t *testing.T) *redisfake.Server {
	srv, err := redisfake.Start()
	require.NoError(t, err)
	redisServer = srv
	t.Cleanup(func() {
		redisServer = nil
		srv.Close()
	})
	return srv
}

// call sends one request as userID (0 = no token) and returns status + body.
func call(t *testing.T, h http.Handler, method, path string, body any, userID int64) (int, []byte) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != 0 {
		token, err := jwtx.NewJWT([]byte(testJwtKey)).Create(&jwtx.UserClaim{UserID: userID, ExpiresAt: time.Now().Add(time.Hour).Unix()})
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes()
}

var userColumns = []string{"UserID", "Username", "Email", "RoleID", "CreatedAt", "UpdatedAt"}

// userRow answers the "Users WHERE UserID = id" lookup.
func userRow(id int64, name, email, role string) sqlfake.Result {
	return sqlfake.Result{
		Match:   fmt.Sprintf(`FROM "Users" WHERE UserID = @p1 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |%d`, id),
		Columns: userColumns,
		Rows:    [][]driver.Value{{id, name, email, role, created, created}},
	}
}

// noRows answers queries containing match with zero rows.
func noRows(match string) sqlfake.Result { return sqlfake.Result{Match: match} }

var timeLiteral = regexp.MustCompile(`'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?'`)

// stableSQL replaces time literals (time.Now() values) with '<time>'.
func stableSQL(sqls []string) []string {
	out := make([]string, len(sqls))
	for i, s := range sqls {
		out[i] = timeLiteral.ReplaceAllString(s, "'<time>'")
	}
	return out
}
