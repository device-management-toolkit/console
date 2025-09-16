package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock database connection.
type MockDB struct {
	mock.Mock
}

func (mdb *MockDB) Open(driverName, dataSourceName string) (*sql.DB, error) {
	args := mdb.Called(driverName, dataSourceName)

	db, ok := args.Get(0).(*sql.DB)
	if !ok {
		return nil, errors.New("failed to cast to *sql.DB")
	}

	return db, args.Error(1)
}

func TestNew_Postgres(t *testing.T) {
	t.Parallel()

	mockDB := new(MockDB)
	mockDB.On("Open", "pgx", "postgres://localhost:5432/testdb").Return(&sql.DB{}, nil)

	db, err := New("postgres://localhost:5432/testdb", mockDB.Open)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.False(t, db.IsEmbedded)
	assert.Equal(t, _defaultMaxPoolSize, db.maxPoolSize)
	assert.Equal(t, _defaultConnAttempts, db.connAttempts)
	assert.Equal(t, _defaultConnTimeout, db.connTimeout)

	mockDB.AssertExpectations(t)
}

func TestNew_Embedded(t *testing.T) {
	t.Parallel()

	mockDB := new(MockDB)
	mockDB.On("Open", "sqlite", mock.Anything).Return(&sql.DB{}, nil)

	db, err := New("sqlite://localhost:5432/testdb", mockDB.Open, EnableForeignKeys(false))
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.True(t, db.IsEmbedded)
	assert.Equal(t, _defaultMaxPoolSize, db.maxPoolSize)
	assert.Equal(t, _defaultConnAttempts, db.connAttempts)
	assert.Equal(t, _defaultConnTimeout, db.connTimeout)

	mockDB.AssertExpectations(t)
}

var ErrTest = errors.New("test error")

func TestCheckNotUnique(t *testing.T) {
	t.Parallel()

	pgErr := &pgconn.PgError{Code: UniqueViolation}

	assert.False(t, CheckNotUnique(ErrTest))
	assert.True(t, CheckNotUnique(pgErr))
}
