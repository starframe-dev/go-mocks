package sqlmock

import (
	"database/sql/driver"
	"fmt"
	"io"
)

// response is the scripted outcome of an expectation.
type response struct {
	rows         *Rows
	lastInsertID int64
	rowsAffected int64
	err          error
}

// Expectation describes one anticipated database operation and its scripted outcome.
type Expectation struct {
	mock      *Mock
	kind      Kind
	query     string
	args      []driver.Value
	checkArgs bool
	consumed  bool
	response  response
}

// WithArgs requires the statement to be issued with exactly these arguments.
// Arguments are normalized like database/sql does (for example int becomes int64).
func (e *Expectation) WithArgs(args ...any) *Expectation {
	normalized := make([]driver.Value, len(args))
	for i, a := range args {
		normalized[i] = normalizeValue(a)
	}

	e.mock.mu.Lock()
	defer e.mock.mu.Unlock()
	e.args = normalized
	e.checkArgs = true
	return e
}

// WillReturnRows sets the rows returned by a query expectation.
func (e *Expectation) WillReturnRows(rows *Rows) *Expectation {
	e.mock.mu.Lock()
	defer e.mock.mu.Unlock()
	e.response.rows = rows
	return e
}

// WillReturnResult sets the result returned by an exec expectation.
func (e *Expectation) WillReturnResult(lastInsertID, rowsAffected int64) *Expectation {
	e.mock.mu.Lock()
	defer e.mock.mu.Unlock()
	e.response.lastInsertID = lastInsertID
	e.response.rowsAffected = rowsAffected
	return e
}

// WillReturnError makes the operation fail with err.
func (e *Expectation) WillReturnError(err error) *Expectation {
	e.mock.mu.Lock()
	defer e.mock.mu.Unlock()
	e.response.err = err
	return e
}

// String describes the expectation for error messages.
func (e *Expectation) String() string {
	if e.query == "" {
		return string(e.kind)
	}
	return fmt.Sprintf("%s %q", e.kind, e.query)
}

// Rows is a scripted query result set.
type Rows struct {
	columns []string
	data    [][]driver.Value
	err     error
}

// NewRows creates an empty result set with the given column names.
func NewRows(columns ...string) *Rows {
	return &Rows{columns: append([]string(nil), columns...)}
}

// AddRow appends a row. A wrong value count or an unsupported value type makes the query fail.
func (r *Rows) AddRow(values ...any) *Rows {
	if len(values) != len(r.columns) {
		r.err = fmt.Errorf("sqlmock: row has %d values but %d columns", len(values), len(r.columns))
		return r
	}

	row := make([]driver.Value, len(values))
	for i, v := range values {
		row[i] = normalizeValue(v)
	}
	r.data = append(r.data, row)
	return r
}

// cursor creates an independent driver-level iterator over a snapshot of the rows.
func (r *Rows) cursor() *cursor {
	data := make([][]driver.Value, len(r.data))
	copy(data, r.data)
	return &cursor{columns: r.columns, data: data}
}

// cursor implements driver.Rows.
type cursor struct {
	columns []string
	data    [][]driver.Value
	pos     int
}

// Columns implements driver.Rows.
func (c *cursor) Columns() []string { return c.columns }

// Close implements driver.Rows.
func (c *cursor) Close() error { return nil }

// Next implements driver.Rows.
func (c *cursor) Next(dest []driver.Value) error {
	if c.pos >= len(c.data) {
		return io.EOF
	}
	copy(dest, c.data[c.pos])
	c.pos++
	return nil
}

// normalizeValue converts v to a driver.Value the way database/sql does; unsupported values are kept as is.
func normalizeValue(v any) driver.Value {
	converted, err := driver.DefaultParameterConverter.ConvertValue(v)
	if err != nil {
		return v
	}
	return converted
}
