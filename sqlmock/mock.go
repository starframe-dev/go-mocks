package sqlmock

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	// ErrUnexpected is returned when the code under test issues a statement nobody expected.
	ErrUnexpected = errors.New("sqlmock: unexpected statement")

	// ErrUnmet is returned by ExpectationsWereMet when some expectations were never used.
	ErrUnmet = errors.New("sqlmock: unmet expectations")
)

// Kind identifies the type of a database operation.
type Kind string

// Operation kinds recorded in history and used by expectations.
const (
	KindQuery    Kind = "query"
	KindExec     Kind = "exec"
	KindBegin    Kind = "begin"
	KindCommit   Kind = "commit"
	KindRollback Kind = "rollback"
)

// Statement is a database operation issued by the code under test.
type Statement struct {
	Kind  Kind
	Query string
	Args  []driver.Value
}

// Mock scripts and records the traffic of a *sql.DB created by New.
//
// Expectations are matched by kind, whitespace-normalized query text and (optionally) arguments.
// Each expectation is used once; for equal statements they are consumed in the order they were added.
type Mock struct {
	mu           sync.Mutex
	expectations []*Expectation
	history      []Statement
}

// New creates a *sql.DB backed by a fresh Mock. No global driver registration is involved.
func New() (*sql.DB, *Mock) {
	m := &Mock{}
	return sql.OpenDB(&connector{mock: m}), m
}

// ExpectQuery registers an expected Query/QueryRow call.
func (m *Mock) ExpectQuery(query string) *Expectation {
	return m.expect(KindQuery, query)
}

// ExpectExec registers an expected Exec call.
func (m *Mock) ExpectExec(query string) *Expectation {
	return m.expect(KindExec, query)
}

// ExpectBegin registers an expected transaction start.
func (m *Mock) ExpectBegin() *Expectation {
	return m.expect(KindBegin, "")
}

// ExpectCommit registers an expected transaction commit.
func (m *Mock) ExpectCommit() *Expectation {
	return m.expect(KindCommit, "")
}

// ExpectRollback registers an expected transaction rollback.
func (m *Mock) ExpectRollback() *Expectation {
	return m.expect(KindRollback, "")
}

// History returns a copy of every statement the code under test issued, including unexpected ones.
func (m *Mock) History() []Statement {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Statement, len(m.history))
	for i, s := range m.history {
		s.Args = append([]driver.Value(nil), s.Args...)
		result[i] = s
	}
	return result
}

// ExpectationsWereMet returns an ErrUnmet-wrapping error if any expectation was never used.
func (m *Mock) ExpectationsWereMet() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var unmet []string
	for _, e := range m.expectations {
		if !e.consumed {
			unmet = append(unmet, e.String())
		}
	}
	if len(unmet) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnmet, strings.Join(unmet, "; "))
}

// Reset removes all expectations and clears the history.
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = nil
	m.history = nil
}

func (m *Mock) expect(kind Kind, query string) *Expectation {
	e := &Expectation{mock: m, kind: kind, query: normalizeQuery(query)}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = append(m.expectations, e)
	return e
}

// consume records the statement and returns the response of the first matching unused expectation.
func (m *Mock) consume(kind Kind, query string, args []driver.Value) (response, error) {
	normalized := normalizeQuery(query)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, Statement{Kind: kind, Query: query, Args: append([]driver.Value(nil), args...)})

	argsMismatch := false
	for _, e := range m.expectations {
		if e.consumed || e.kind != kind || e.query != normalized {
			continue
		}
		if e.checkArgs && !argsEqual(e.args, args) {
			argsMismatch = true
			continue
		}
		e.consumed = true
		return e.response, nil
	}

	if argsMismatch {
		return response{}, fmt.Errorf("%w: %s %q called with arguments %v that match no expectation", ErrUnexpected, kind, normalized, args)
	}
	return response{}, fmt.Errorf("%w: %s %q", ErrUnexpected, kind, normalized)
}

// argsEqual compares argument lists, treating nil and empty slices as equal.
func argsEqual(expected, actual []driver.Value) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !reflect.DeepEqual(expected[i], actual[i]) {
			return false
		}
	}
	return true
}

// normalizeQuery collapses all whitespace so formatting differences do not break matching.
func normalizeQuery(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
