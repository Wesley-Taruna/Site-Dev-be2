package repository

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/db"
	"gitlab.com/fami-app-v2/familib/familabs/env"
	"gitlab.com/fami-app-v2/familib/familabs/opt"
	"gorm.io/gorm"
)

// ErrStoreDBConfig: [store_database] has a table or column name that is not
// a plain SQL name, or an unknown driver. Nothing is sent to a database.
var ErrStoreDBConfig = errors.New("store_database: invalid config")

// ErrStoreDBUnavailable: the separate store database cannot be opened now
// (retried after storeDBRetryAfter).
var ErrStoreDBUnavailable = errors.New("store_database: cannot connect")

// IFamilyMartStoreRepository reads the active FamilyMart stores inside a
// lat/lng box (GET /v1/site/familymart-stores, Wes 29 Oct). Where from:
// [store_database] in the config. host empty (default) = the main database
// connection, table AMFIBI_MASTER.dbo.MStore, exactly what /v1/site/maps and
// nearest_familymart already read. host set = a separate SQL Server (or
// MySQL) connection opened on first use. Table and column names can be
// changed there too.
type IFamilyMartStoreRepository interface {
	// InBox returns at most limit active stores with
	// minLat <= lat <= maxLat and minLng <= lng <= maxLng (no order).
	InBox(ctx context.Context, minLat, maxLat, minLng, maxLng float64, limit int) (stores []model.FamilyMartStore, err error)
	// CacheTag changes when the data source (host/database/table/columns)
	// changes, so cached rows of another source are not used.
	CacheTag() string
}

// StoreDBConfig is [store_database]. Credentials are only ever read from
// config/env.toml (not tracked by git).
type StoreDBConfig struct {
	Driver   string // "sqlserver" (default) or "mysql"
	Host     string // "" = use the main database connection
	Port     int
	Username string
	Password string
	Database string
	Encrypt  string // SQL Server "encrypt" option ("" = driver default)

	Table         string // default AMFIBI_MASTER.dbo.MStore
	CodeColumn    string // default StoreCode
	NameColumn    string // default StoreName
	LatColumn     string // default Latitude
	LngColumn     string // default Longitude
	AddressColumn string // default "" = no address in the answer
	ActiveColumn  string // default IsActive (= 1); "" = every row
}

// LoadStoreDBConfig reads [store_database] with its defaults.
func LoadStoreDBConfig(e env.Provider) StoreDBConfig {
	c := StoreDBConfig{
		Driver:       "sqlserver",
		Table:        "AMFIBI_MASTER.dbo.MStore",
		CodeColumn:   "StoreCode",
		NameColumn:   "StoreName",
		LatColumn:    "Latitude",
		LngColumn:    "Longitude",
		ActiveColumn: "IsActive",
	}
	if e == nil {
		return c
	}
	str := func(key string, dst *string) {
		if v := strings.TrimSpace(e.GetString("store_database." + key)); v != "" {
			*dst = v
		}
	}
	str("driver", &c.Driver)
	str("host", &c.Host)
	str("username", &c.Username)
	c.Password = e.GetString("store_database.password")
	str("database", &c.Database)
	str("encrypt", &c.Encrypt)
	str("table", &c.Table)
	str("code_column", &c.CodeColumn)
	str("name_column", &c.NameColumn)
	str("lat_column", &c.LatColumn)
	str("lng_column", &c.LngColumn)
	str("address_column", &c.AddressColumn)
	// active_column may be set to "" on purpose (no active flag)
	if e.IsSet("store_database.active_column") {
		c.ActiveColumn = strings.TrimSpace(e.GetString("store_database.active_column"))
	}
	c.Driver = strings.ToLower(c.Driver)
	c.Port = e.GetInt("store_database.port")
	if c.Port == 0 {
		if c.Driver == "mysql" {
			c.Port = 3306
		} else {
			c.Port = 1433
		}
	}
	return c
}

// Separate reports whether a separate store database is configured.
func (c StoreDBConfig) Separate() bool { return c.Host != "" }

var (
	// sqlName: Column, or db.schema.Table (letters, digits, _; max 3 parts)
	sqlColumnName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
	sqlTableName  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}(\.[A-Za-z_][A-Za-z0-9_]{0,127}){0,2}$`)
)

// Validate refuses anything that is not a plain SQL name (the names are
// put into the query text, so this is the injection guard).
func (c StoreDBConfig) Validate() error {
	if c.Driver != "sqlserver" && c.Driver != "mysql" {
		return fmt.Errorf("%w: driver %q (sqlserver or mysql)", ErrStoreDBConfig, c.Driver)
	}
	if !sqlTableName.MatchString(c.Table) {
		return fmt.Errorf("%w: table %q", ErrStoreDBConfig, c.Table)
	}
	cols := map[string]string{"code_column": c.CodeColumn, "name_column": c.NameColumn, "lat_column": c.LatColumn, "lng_column": c.LngColumn}
	if c.AddressColumn != "" {
		cols["address_column"] = c.AddressColumn
	}
	if c.ActiveColumn != "" {
		cols["active_column"] = c.ActiveColumn
	}
	for k, v := range cols {
		if !sqlColumnName.MatchString(v) {
			return fmt.Errorf("%w: %s %q", ErrStoreDBConfig, k, v)
		}
	}
	return nil
}

// Query is the box query with ? placeholders: minLat, maxLat, minLng,
// maxLng. Same filters as the existing store queries (coordinates set,
// active, not 'FOODTRUCK 1').
func (c StoreDBConfig) Query(limit int) string {
	cols := fmt.Sprintf("%s AS StoreCode, %s AS StoreName, %s AS Latitude, %s AS Longitude", c.CodeColumn, c.NameColumn, c.LatColumn, c.LngColumn)
	if c.AddressColumn != "" {
		cols += fmt.Sprintf(", %s AS Address", c.AddressColumn)
	}
	where := fmt.Sprintf("%s BETWEEN ? AND ? AND %s BETWEEN ? AND ? AND %s <> 'FOODTRUCK 1'", c.LatColumn, c.LngColumn, c.NameColumn)
	if c.ActiveColumn != "" {
		where += fmt.Sprintf(" AND %s = 1", c.ActiveColumn)
	}
	if c.Driver == "mysql" {
		return fmt.Sprintf("SELECT %s FROM %s WHERE %s LIMIT %d", cols, c.Table, where, limit)
	}
	return fmt.Sprintf("SELECT TOP (%d) %s FROM %s WHERE %s", limit, cols, c.Table, where)
}

// tag is a short fingerprint of the source (no password in it).
func (c StoreDBConfig) tag() string {
	h := sha1.Sum([]byte(strings.Join([]string{c.Driver, c.Host, fmt.Sprint(c.Port), c.Database, c.Table,
		c.CodeColumn, c.NameColumn, c.LatColumn, c.LngColumn, c.AddressColumn, c.ActiveColumn}, "|")))
	return hex.EncodeToString(h[:4])
}

// storeDBRetryAfter: after a failed connect, wait this long before trying
// the separate store database again (each request in between gets 503).
const storeDBRetryAfter = time.Minute

// storeDBConn is the process-wide separate connection (opened once).
type storeDBConn struct {
	mu       sync.Mutex
	db       *gorm.DB
	failedAt time.Time
}

var sharedStoreDB = &storeDBConn{}

type FamilyMartStoreRepository struct {
	opt  opt.Options
	cfg  StoreDBConfig
	conn *storeDBConn
}

func NewFamilyMartStoreRepository() IFamilyMartStoreRepository {
	return &FamilyMartStoreRepository{
		opt:  global.Setup.Common,
		cfg:  LoadStoreDBConfig(global.Setup.Common.Env),
		conn: sharedStoreDB,
	}
}

func (r *FamilyMartStoreRepository) CacheTag() string { return r.cfg.tag() }

func (r *FamilyMartStoreRepository) InBox(ctx context.Context, minLat, maxLat, minLng, maxLng float64, limit int) (stores []model.FamilyMartStore, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "FamilyMartStoreRepository.InBox")
	defer span.End()

	if err = r.cfg.Validate(); err != nil {
		span.RecordError(err)
		return nil, err
	}
	conn, err := r.database(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if err = conn.Raw(r.cfg.Query(limit), minLat, maxLat, minLng, maxLng).Scan(&stores).Error; err != nil {
		span.RecordError(err)
		return nil, err
	}
	return stores, nil
}

// database is the main read connection, or the separate store database
// when [store_database] host is set.
func (r *FamilyMartStoreRepository) database(ctx context.Context) (*gorm.DB, error) {
	if !r.cfg.Separate() {
		return readDB(ctx, r.opt), nil
	}
	r.conn.mu.Lock()
	defer r.conn.mu.Unlock()
	if r.conn.db != nil {
		return r.conn.db.WithContext(ctx), nil
	}
	if !r.conn.failedAt.IsZero() && time.Since(r.conn.failedAt) < storeDBRetryAfter {
		return nil, ErrStoreDBUnavailable
	}
	conn, err := r.open()
	if err != nil {
		r.conn.failedAt = time.Now()
		// the error text of the drivers never carries the password, but
		// keep only host/port/database in what is logged
		return nil, fmt.Errorf("%w (%s %s:%d/%s): %v", ErrStoreDBUnavailable, r.cfg.Driver, r.cfg.Host, r.cfg.Port, r.cfg.Database, redact(err, r.cfg.Password))
	}
	r.conn.db, r.conn.failedAt = conn, time.Time{}
	return conn.WithContext(ctx), nil
}

func (r *FamilyMartStoreRepository) open() (*gorm.DB, error) {
	name := r.opt.Config.App.Name
	lgr := r.opt.Logger.Logger()
	var (
		conn *gorm.DB
		err  error
	)
	switch r.cfg.Driver {
	case "mysql":
		conn, err = db.NewMysqlDatabase(name, lgr, "store_database", db.DBMysqlOption{
			Host: r.cfg.Host, Port: r.cfg.Port, Username: r.cfg.Username, Password: r.cfg.Password, Name: r.cfg.Database,
			AdditionalParameters: "parseTime=true&timeout=15s", MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Hour,
		})
	default:
		conn, err = db.NewSqlServerDatabase(name, lgr, "store_database", db.DBMssqlOption{
			Host: r.cfg.Host, Port: r.cfg.Port, Username: r.cfg.Username, Password: r.cfg.Password, DbName: r.cfg.Database,
			Encrypt: r.cfg.Encrypt, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Hour, ConnTimeout: 15,
		})
	}
	if err != nil {
		return nil, err
	}
	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	if err = sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return conn, nil
}

// redact removes the password from an error text, just in case.
func redact(err error, secret string) string {
	s := err.Error()
	if secret != "" {
		s = strings.ReplaceAll(s, secret, "***")
	}
	return s
}
