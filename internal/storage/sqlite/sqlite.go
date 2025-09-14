package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"modernc.org/sqlite"
	"strings"
	"urlshortener/internal/storage"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	// Для modernc.org/sqlite
	dsn := "file:" + storagePath + "?cache=shared&mode=rwc"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Создаем таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS url(
			id INTEGER PRIMARY KEY,
			alias TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_alias ON url(alias)
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.Prepare("INSERT INTO url(url, alias) VALUES(?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {
		// Проверяем ошибку уникальности для modernc.org/sqlite
		if sqliteErr, ok := err.(*sqlite.Error); ok {
			if sqliteErr.Code() == 2067 { // SQLITE_CONSTRAINT_UNIQUE
				return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
			}
		}

		// Дополнительная проверка по тексту ошибки
		if strings.Contains(err.Error(), "constraint failed") {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare("SELECT url FROM url WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var resURL string

	err = stmt.QueryRow(alias).Scan(&resURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}

		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare("DELETE FROM url WHERE alias = ?")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: get rows affected: %w", op, err)
	}

	if rowsAffected == 0 {
		return storage.ErrURLNotFound
	}

	return nil
}
