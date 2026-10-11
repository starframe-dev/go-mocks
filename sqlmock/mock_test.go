package sqlmock

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const selectUser = "SELECT id, name FROM users WHERE id = ?"

type user struct {
	ID   int
	Name string
}

func newMock(t *testing.T) (*sql.DB, *Mock) {
	t.Helper()
	db, m := New()
	t.Cleanup(func() { _ = db.Close() })
	return db, m
}

func TestMock_QueryReturnsScriptedRows(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery(selectUser).WithArgs(1).
		WillReturnRows(NewRows("id", "name").AddRow(1, "anya").AddRow(2, "nea"))

	rows, err := db.Query(selectUser, 1)
	require.NoError(t, err)
	defer rows.Close()

	var got []user
	for rows.Next() {
		var u user
		require.NoError(t, rows.Scan(&u.ID, &u.Name))
		got = append(got, u)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []user{{1, "anya"}, {2, "nea"}}, got)
	assert.NoError(t, m.ExpectationsWereMet())
}

func TestMock_QueryRow(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery(selectUser).WillReturnRows(NewRows("id", "name").AddRow(7, "x"))

	var u user
	err := db.QueryRow(selectUser, 7).Scan(&u.ID, &u.Name)

	require.NoError(t, err)
	assert.Equal(t, user{7, "x"}, u)
}

func TestMock_QueryRowNoRows(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery(selectUser).WillReturnRows(NewRows("id", "name"))

	var u user
	err := db.QueryRow(selectUser, 1).Scan(&u.ID, &u.Name)

	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestMock_QueryWithoutRowsReturnsEmptySet(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT 1")

	rows, err := db.Query("SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	assert.False(t, rows.Next())
}

func TestMock_QueryMatchingIgnoresWhitespace(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT   id,\n\t name FROM users").WillReturnRows(NewRows("id", "name"))

	rows, err := db.Query("SELECT id, name\nFROM users")

	require.NoError(t, err)
	require.NoError(t, rows.Close())
}

func TestMock_ExecReturnsScriptedResult(t *testing.T) {
	db, m := newMock(t)
	m.ExpectExec("INSERT INTO users (name) VALUES (?)").WithArgs("anya").WillReturnResult(5, 1)

	res, err := db.Exec("INSERT INTO users (name) VALUES (?)", "anya")
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)
	assert.EqualValues(t, 5, id)
	assert.EqualValues(t, 1, affected)
	assert.NoError(t, m.ExpectationsWereMet())
}

func TestMock_ScriptedErrors(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT 1").WillReturnError(assert.AnError)
	m.ExpectExec("DELETE FROM t").WillReturnError(assert.AnError)

	_, queryErr := db.Query("SELECT 1")
	_, execErr := db.Exec("DELETE FROM t")

	assert.ErrorIs(t, queryErr, assert.AnError)
	assert.ErrorIs(t, execErr, assert.AnError)
	assert.NoError(t, m.ExpectationsWereMet())
}

func TestMock_UnexpectedStatement(t *testing.T) {
	db, _ := newMock(t)

	_, err := db.Exec("DROP TABLE users")

	assert.ErrorIs(t, err, ErrUnexpected)
	assert.ErrorContains(t, err, "DROP TABLE users")
}

func TestMock_KindMustMatch(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT 1")

	_, err := db.Exec("SELECT 1")

	assert.ErrorIs(t, err, ErrUnexpected)
}

func TestMock_ArgsMismatch(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery(selectUser).WithArgs(1).WillReturnRows(NewRows("id", "name"))

	_, err := db.Query(selectUser, 2)

	require.ErrorIs(t, err, ErrUnexpected)
	assert.ErrorContains(t, err, "arguments")
	assert.ErrorIs(t, m.ExpectationsWereMet(), ErrUnmet)
}

func TestMock_ArgsAreNormalized(t *testing.T) {
	db, m := newMock(t)
	when := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	m.ExpectExec("UPDATE t SET a=?, b=?, c=?, d=?").
		WithArgs(1, int32(2), true, when).
		WillReturnResult(0, 1)

	_, err := db.Exec("UPDATE t SET a=?, b=?, c=?, d=?", int8(1), 2, true, when)

	assert.NoError(t, err)
}

func TestMock_ExpectWithoutArgsAcceptsAny(t *testing.T) {
	db, m := newMock(t)
	m.ExpectExec("DELETE FROM t WHERE id = ?")

	_, err := db.Exec("DELETE FROM t WHERE id = ?", 99)

	assert.NoError(t, err)
}

func TestMock_ExpectEmptyArgs(t *testing.T) {
	db, m := newMock(t)
	m.ExpectExec("DELETE FROM t").WithArgs()

	_, err := db.Exec("DELETE FROM t")

	assert.NoError(t, err)
}

func TestMock_ExpectationsConsumedInOrder(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT n").WillReturnRows(NewRows("n").AddRow(1))
	m.ExpectQuery("SELECT n").WillReturnRows(NewRows("n").AddRow(2))

	var first, second, third int
	require.NoError(t, db.QueryRow("SELECT n").Scan(&first))
	require.NoError(t, db.QueryRow("SELECT n").Scan(&second))
	err := db.QueryRow("SELECT n").Scan(&third)

	assert.Equal(t, 1, first)
	assert.Equal(t, 2, second)
	assert.ErrorIs(t, err, ErrUnexpected)
}

func TestMock_RowShapeErrors(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT a, b").WillReturnRows(NewRows("a", "b").AddRow(1))

	_, err := db.Query("SELECT a, b")

	assert.ErrorContains(t, err, "1 values but 2 columns")
}

func TestMock_ExpectationsWereMet(t *testing.T) {
	_, m := newMock(t)
	m.ExpectQuery("SELECT 1")
	m.ExpectBegin()

	err := m.ExpectationsWereMet()

	require.ErrorIs(t, err, ErrUnmet)
	assert.ErrorContains(t, err, `query "SELECT 1"`)
	assert.ErrorContains(t, err, "begin")
}

func TestMock_History(t *testing.T) {
	db, m := newMock(t)
	m.ExpectExec("INSERT INTO t VALUES (?, ?)")

	_, _ = db.Exec("INSERT INTO t VALUES (?, ?)", 1, "a")
	_, _ = db.Exec("UNEXPECTED")

	history := m.History()
	require.Len(t, history, 2)
	assert.Equal(t, KindExec, history[0].Kind)
	assert.Equal(t, "INSERT INTO t VALUES (?, ?)", history[0].Query)
	assert.Equal(t, []any{int64(1), "a"}, toAny(history[0].Args))
	assert.Equal(t, "UNEXPECTED", history[1].Query)

	history[0].Args[0] = "mutated"
	assert.EqualValues(t, 1, m.History()[0].Args[0])
}

func TestMock_Reset(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery("SELECT 1")
	_, _ = db.Query("SELECT 1")

	m.Reset()

	assert.Empty(t, m.History())
	assert.NoError(t, m.ExpectationsWereMet())
}

func TestMock_TransactionCommit(t *testing.T) {
	db, m := newMock(t)
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO t VALUES (1)").WillReturnResult(1, 1)
	m.ExpectCommit()

	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec("INSERT INTO t VALUES (1)")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.NoError(t, m.ExpectationsWereMet())
	kinds := []Kind{}
	for _, s := range m.History() {
		kinds = append(kinds, s.Kind)
	}
	assert.Equal(t, []Kind{KindBegin, KindExec, KindCommit}, kinds)
}

func TestMock_TransactionRollback(t *testing.T) {
	db, m := newMock(t)
	m.ExpectBegin()
	m.ExpectRollback()

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	assert.NoError(t, m.ExpectationsWereMet())
}

func TestMock_TransactionErrors(t *testing.T) {
	db, m := newMock(t)
	m.ExpectBegin().WillReturnError(assert.AnError)

	_, err := db.Begin()
	assert.ErrorIs(t, err, assert.AnError)

	m.ExpectBegin()
	m.ExpectCommit().WillReturnError(assert.AnError)
	tx, err := db.Begin()
	require.NoError(t, err)
	assert.ErrorIs(t, tx.Commit(), assert.AnError)
}

func TestMock_UnexpectedBegin(t *testing.T) {
	db, _ := newMock(t)

	_, err := db.Begin()

	assert.ErrorIs(t, err, ErrUnexpected)
}

func TestMock_PreparedStatement(t *testing.T) {
	db, m := newMock(t)
	m.ExpectQuery(selectUser).WithArgs(3).WillReturnRows(NewRows("id", "name").AddRow(3, "p"))
	m.ExpectExec("DELETE FROM t WHERE id = ?").WithArgs(3).WillReturnResult(0, 2)

	q, err := db.Prepare(selectUser)
	require.NoError(t, err)
	var u user
	require.NoError(t, q.QueryRow(3).Scan(&u.ID, &u.Name))
	assert.Equal(t, user{3, "p"}, u)

	e, err := db.Prepare("DELETE FROM t WHERE id = ?")
	require.NoError(t, err)
	res, err := e.Exec(3)
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)

	assert.EqualValues(t, 2, affected)
	assert.NoError(t, m.ExpectationsWereMet())
}

func TestMock_Ping(t *testing.T) {
	db, _ := newMock(t)

	assert.NoError(t, db.Ping())
}

func TestMock_ConcurrentAccess(t *testing.T) {
	db, m := newMock(t)
	const workers = 20
	for i := 0; i < workers; i++ {
		m.ExpectExec("UPDATE counters SET n = n + 1").WillReturnResult(0, 1)
	}

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := db.Exec("UPDATE counters SET n = n + 1")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
	assert.NoError(t, m.ExpectationsWereMet())
	assert.Len(t, m.History(), workers)
}

func TestExpectation_String(t *testing.T) {
	_, m := newMock(t)

	assert.Equal(t, `query "SELECT 1"`, m.ExpectQuery(" SELECT  1 ").String())
	assert.Equal(t, "commit", m.ExpectCommit().String())
}

func TestNewRows_ColumnsAreCopied(t *testing.T) {
	columns := []string{"a"}
	rows := NewRows(columns...)

	columns[0] = "mutated"

	assert.Equal(t, []string{"a"}, rows.cursor().Columns())
}

func toAny[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
