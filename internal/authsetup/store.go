package authsetup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Required to register the SQLite driver with database/sql.
	_ "modernc.org/sqlite"
)

const (
	setupTokenLifetime = 10 * time.Minute
	setupTokenSize     = 32
	stateDirMode       = 0o700
	stateFileMode      = 0o600
)

type stateStore struct {
	db *sql.DB
}

func openStateStore(path string) (*stateStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), stateDirMode); err != nil {
		return nil, fmt.Errorf("create auth state directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, stateFileMode)
	if err != nil {
		return nil, fmt.Errorf("create auth state database: %w", err)
	}

	if err := file.Chmod(stateFileMode); err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("restrict auth state database permissions: %w", err)
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close auth state database file: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve auth state database path: %w", err)
	}

	urlPath := filepath.ToSlash(absPath)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	dsn := url.URL{Scheme: "file", Path: urlPath}
	query := dsn.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	dsn.RawQuery = query.Encode()

	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open auth state database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS auth_setup (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		token TEXT NOT NULL DEFAULT '',
		token_hash BLOB NOT NULL DEFAULT X'',
		expires_at INTEGER NOT NULL DEFAULT 0,
		auth_version INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("initialize auth setup state: %w", err)
	}

	if err := os.Chmod(path, stateFileMode); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("restrict auth state database permissions: %w", err)
	}

	if _, err := db.ExecContext(context.Background(), `INSERT OR IGNORE INTO auth_setup (id) VALUES (1)`); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("initialize auth setup state row: %w", err)
	}

	return &stateStore{db: db}, nil
}

func (s *stateStore) close() error {
	return s.db.Close()
}

func (s *stateStore) transaction(fn func(*sql.Conn) error) error {
	conn, err := s.db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("open auth state connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("lock auth state: %w", err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()

	if err := fn(conn); err != nil {
		return err
	}

	if _, err := conn.ExecContext(context.Background(), "COMMIT"); err != nil {
		return fmt.Errorf("commit auth state: %w", err)
	}

	return nil
}

func (s *stateStore) setupToken(now time.Time) (string, error) {
	var token string

	err := s.transaction(func(conn *sql.Conn) error {
		var current string

		var expiresAt int64
		if err := conn.QueryRowContext(context.Background(), `SELECT token, expires_at FROM auth_setup WHERE id = 1`).
			Scan(&current, &expiresAt); err != nil {
			return fmt.Errorf("read auth setup token: %w", err)
		}

		if current != "" && now.Unix() < expiresAt {
			token = current

			return nil
		}

		randomToken := make([]byte, setupTokenSize)
		if _, err := rand.Read(randomToken); err != nil {
			return fmt.Errorf("generate auth setup token: %w", err)
		}

		token = base64.RawURLEncoding.EncodeToString(randomToken)

		hash := sha256.Sum256([]byte(token))
		if _, err := conn.ExecContext(context.Background(), `UPDATE auth_setup SET token = ?, token_hash = ?, expires_at = ? WHERE id = 1`,
			token, hash[:], now.Add(setupTokenLifetime).Unix()); err != nil {
			return fmt.Errorf("persist auth setup token: %w", err)
		}

		return nil
	})

	return token, err
}

func (s *stateStore) consumeSetupToken(token string, now time.Time, alreadyConfigured func() (bool, error), setup func() error) error {
	return s.transaction(func(conn *sql.Conn) error {
		configured, err := alreadyConfigured()
		if err != nil {
			return err
		}

		if configured {
			return ErrAlreadyConfigured
		}

		var storedHash []byte

		var expiresAt int64
		if err := conn.QueryRowContext(context.Background(), `SELECT token_hash, expires_at FROM auth_setup WHERE id = 1`).
			Scan(&storedHash, &expiresAt); err != nil {
			return fmt.Errorf("read auth setup token: %w", err)
		}

		providedHash := sha256.Sum256([]byte(token))
		if expiresAt <= now.Unix() || len(storedHash) != len(providedHash) ||
			subtle.ConstantTimeCompare(storedHash, providedHash[:]) != 1 {
			return ErrInvalidSetupToken
		}

		if err := setup(); err != nil {
			return err
		}

		if _, err := conn.ExecContext(context.Background(), `UPDATE auth_setup SET token = '', token_hash = X'', expires_at = 0 WHERE id = 1`); err != nil {
			return fmt.Errorf("consume auth setup token: %w", err)
		}

		return nil
	})
}

func (s *stateStore) authVersion() (uint64, error) {
	var version uint64
	if err := s.db.QueryRowContext(context.Background(), `SELECT auth_version FROM auth_setup WHERE id = 1`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read auth version: %w", err)
	}

	return version, nil
}

func (s *stateStore) withAuthVersion(fn func(uint64) (uint64, error)) error {
	return s.transaction(func(conn *sql.Conn) error {
		var current uint64
		if err := conn.QueryRowContext(context.Background(), `SELECT auth_version FROM auth_setup WHERE id = 1`).
			Scan(&current); err != nil {
			return fmt.Errorf("read auth version: %w", err)
		}

		next, err := fn(current)
		if err != nil {
			return err
		}

		if next < current {
			return errAuthVersionCannotDecrease
		}

		if next != current {
			if _, err := conn.ExecContext(context.Background(), `UPDATE auth_setup SET auth_version = ? WHERE id = 1`, next); err != nil {
				return fmt.Errorf("persist auth version: %w", err)
			}
		}

		return nil
	})
}

var ErrInvalidSetupToken = errors.New("invalid or expired setup token")
