// Package sqlfake is a tiny in-memory database/sql driver for tests. It
// answers queries with canned rows and records every query, so repository
// and end-to-end tests can run the real gorm + SQL Server dialect code
// without a database. Do not use it outside tests.
package sqlfake

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

// Result is the canned answer for queries whose text contains Match. The
// text is the SQL (with @p1.. placeholders) followed by " |<value>" for
// every argument, so Match can also pick a query by its values.
type Result struct {
	Match   string
	Columns []string
	Rows    [][]driver.Value
	Err     error
}

// DB records queries and answers them from Results (first match wins;
// no match = zero rows).
type DB struct {
	mu      sync.Mutex
	Results []Result
	queries []string
}

// Queries returns the SQL text of every query run so far.
func (d *DB) Queries() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.queries...)
}

var (
	counter  int64
	registry sync.Map
)

func init() {
	sql.Register("sqlfake", drv{})
}

// Open returns a *sql.DB backed by d.
func Open(d *DB) (*sql.DB, error) {
	name := fmt.Sprintf("db%d", atomic.AddInt64(&counter, 1))
	registry.Store(name, d)
	return sql.Open("sqlfake", name)
}

type drv struct{}

func (drv) Open(name string) (driver.Conn, error) {
	v, ok := registry.Load(name)
	if !ok {
		return nil, errors.New("sqlfake: unknown db " + name)
	}
	return &conn{db: v.(*DB)}, nil
}

type conn struct{ db *DB }

func (c *conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("sqlfake: prepare not supported")
}
func (c *conn) Close() error              { return nil }
func (c *conn) Begin() (driver.Tx, error) { return nil, errors.New("sqlfake: no transactions") }

func (c *conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	text := query
	for _, a := range args {
		text += fmt.Sprintf(" |%v", a.Value)
	}

	c.db.mu.Lock()
	c.db.queries = append(c.db.queries, query)
	results := c.db.Results
	c.db.mu.Unlock()

	for _, r := range results {
		if strings.Contains(text, r.Match) {
			if r.Err != nil {
				return nil, r.Err
			}
			return &rows{columns: r.Columns, data: r.Rows}, nil
		}
	}
	return &rows{}, nil
}

type rows struct {
	columns []string
	data    [][]driver.Value
	i       int
}

func (r *rows) Columns() []string { return r.columns }
func (r *rows) Close() error      { return nil }
func (r *rows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}
