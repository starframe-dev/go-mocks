package sqlmock

import (
	"context"
	"database/sql/driver"
)

// Compile-time checks that the driver glue satisfies the interfaces database/sql probes for.
var (
	_ driver.Connector      = (*connector)(nil)
	_ driver.Driver         = (*connector)(nil)
	_ driver.Conn           = (*conn)(nil)
	_ driver.ConnBeginTx    = (*conn)(nil)
	_ driver.QueryerContext = (*conn)(nil)
	_ driver.ExecerContext  = (*conn)(nil)
	_ driver.Stmt           = (*stmt)(nil)
	_ driver.Tx             = (*tx)(nil)
	_ driver.Result         = result{}
)

// connector hands out connections that all share one Mock.
type connector struct {
	mock *Mock
}

// Connect implements driver.Connector.
func (c *connector) Connect(context.Context) (driver.Conn, error) {
	return &conn{mock: c.mock}, nil
}

// Driver implements driver.Connector.
func (c *connector) Driver() driver.Driver { return c }

// Open implements driver.Driver.
func (c *connector) Open(string) (driver.Conn, error) {
	return &conn{mock: c.mock}, nil
}

// conn is a driver connection that routes everything to the Mock.
type conn struct {
	mock *Mock
}

// Prepare implements driver.Conn.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return &stmt{mock: c.mock, query: query}, nil
}

// Close implements driver.Conn.
func (c *conn) Close() error { return nil }

// Begin implements driver.Conn.
func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// BeginTx implements driver.ConnBeginTx.
func (c *conn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if _, err := c.mock.run(KindBegin, "", nil); err != nil {
		return nil, err
	}
	return &tx{mock: c.mock}, nil
}

// QueryContext implements driver.QueryerContext.
func (c *conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.mock.query(query, namedToValues(args))
}

// ExecContext implements driver.ExecerContext.
func (c *conn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.mock.exec(query, namedToValues(args))
}

// stmt is a prepared statement; it only remembers the query text.
type stmt struct {
	mock  *Mock
	query string
}

// Close implements driver.Stmt.
func (s *stmt) Close() error { return nil }

// NumInput implements driver.Stmt. -1 disables argument count checking.
func (s *stmt) NumInput() int { return -1 }

// Exec implements driver.Stmt.
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.mock.exec(s.query, args)
}

// Query implements driver.Stmt.
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.mock.query(s.query, args)
}

// tx is a driver transaction.
type tx struct {
	mock *Mock
}

// Commit implements driver.Tx.
func (t *tx) Commit() error {
	_, err := t.mock.run(KindCommit, "", nil)
	return err
}

// Rollback implements driver.Tx.
func (t *tx) Rollback() error {
	_, err := t.mock.run(KindRollback, "", nil)
	return err
}

// result implements driver.Result.
type result struct {
	lastInsertID int64
	rowsAffected int64
}

// LastInsertId implements driver.Result.
func (r result) LastInsertId() (int64, error) { return r.lastInsertID, nil }

// RowsAffected implements driver.Result.
func (r result) RowsAffected() (int64, error) { return r.rowsAffected, nil }

// run consumes an expectation and surfaces its scripted error, if any.
func (m *Mock) run(kind Kind, query string, args []driver.Value) (response, error) {
	resp, err := m.consume(kind, query, args)
	if err != nil {
		return response{}, err
	}
	if resp.err != nil {
		return response{}, resp.err
	}
	return resp, nil
}

func (m *Mock) query(query string, args []driver.Value) (driver.Rows, error) {
	resp, err := m.run(KindQuery, query, args)
	if err != nil {
		return nil, err
	}
	if resp.rows == nil {
		return NewRows().cursor(), nil
	}
	if resp.rows.err != nil {
		return nil, resp.rows.err
	}
	return resp.rows.cursor(), nil
}

func (m *Mock) exec(query string, args []driver.Value) (driver.Result, error) {
	resp, err := m.run(KindExec, query, args)
	if err != nil {
		return nil, err
	}
	return result{lastInsertID: resp.lastInsertID, rowsAffected: resp.rowsAffected}, nil
}

// namedToValues drops argument names and ordinals; only positional values are matched.
func namedToValues(args []driver.NamedValue) []driver.Value {
	values := make([]driver.Value, len(args))
	for i, a := range args {
		values[i] = a.Value
	}
	return values
}
