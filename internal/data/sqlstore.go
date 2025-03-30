package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

/*
A SQLStore manages Documents via sqlite
*/

// to avoid concurrency issues with background saves
var mutex sync.Mutex

var schema string = `
	CREATE TABLE document (
		id INTEGER PRIMARY KEY,
		in_trash INTEGER,
		name TEXT,
		contents TEXT,
		created_date TEXT,
		updated_date TEXT
 	);
	CREATE TABLE config (
		key TEXT UNIQUE,
		value TEXT
	);
	`

var LAST_OPENED = "last_opened_key"

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore() *SQLStore {
	s := &SQLStore{}
	return s
}

func (s *SQLStore) Open(filepath string) error {
	c, err := sql.Open("sqlite", filepath)
	if err != nil {
		return err
	}
	s.db = c
	return nil
}

func (s *SQLStore) Create(filepath string) error {
	err := s.Open(filepath)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(schema)
	if err != nil {
		return err
	}
	return nil
}

// Return a slice of DocReferences for each Document in the Store (may return an empty list)
// Optionally toggle whether to look in trash or not
func (s *SQLStore) ListDocuments(t bool) ([]DocReference, error) {
	if s.db == nil {
		return nil, errors.New("Cannot list documents- must open this SQLStore first.")
	}

	flag := 0
	if t {
		flag = 1
	}
	query := fmt.Sprintf("SELECT id, name FROM document where in_trash = %d", flag)

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DocReference, 0)
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			return nil, err
		} else {
			result = append(result, DocReference{id, name})
		}
	}
	return result, nil
}

func (s *SQLStore) CreateDocument(name string, text string) (int64, error) {
	if s.db == nil {
		return 0, errors.New("Cannot save document-  must open this SQLStore first.")
	}
	stmt, err := s.db.Prepare("INSERT INTO document(in_trash, name, contents, created_date, updated_date) VALUES (?, ?, ?, ?, ?) RETURNING id")
	if err != nil {
		return 0, err
	}
	now := s.timeNow()
	mutex.Lock()
	result, err := stmt.Exec(false, name, text, now, now)
	mutex.Unlock()
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *SQLStore) SaveDocument(key string, text string) error {
	if s.db == nil {
		return errors.New("Cannot save document-  must open this SQLStore first.")
	}
	stmt, err := s.db.Prepare("UPDATE document SET contents = ?, updated_date = ? WHERE id = ?")
	if err != nil {
		return err
	}
	now := s.timeNow()
	mutex.Lock()
	_, err = stmt.Exec(text, now, key)
	mutex.Unlock()
	defer stmt.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) LoadDocument(key string) (string, error) {
	var result string
	if s.db == nil {
		return "", errors.New("Cannot load document-  must open this SQLStore first.")
	}
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	row := tx.QueryRow("SELECT contents FROM document WHERE id = ?", key)
	err = row.Scan(&result)
	if err != nil {
		tx.Rollback()
		return "", nil
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO config(key, value) VALUES ($1, $2) ON CONFLICT(key) DO UPDATE SET value=$2",
		LAST_OPENED, key)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	err = tx.Commit()
	if err != nil {
		return "", err
	}
	return result, nil
}

func (s *SQLStore) DeleteDocument(key string) error {
	if s.db == nil {
		return errors.New("Cannot delete document-  must open this SQLStore first.")
	}
	// Deleting a Document is full removal
	stmt, err := s.db.Prepare("DELETE FROM document WHERE id = ?")
	if err != nil {
		return err
	}
	mutex.Lock()
	_, err = stmt.Exec(key)
	mutex.Unlock()
	defer stmt.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) TrashDocument(key string) error {
	if s.db == nil {
		return errors.New("Cannot trash document-  must open this SQLStore first.")
	}
	// Trashing a Document is just marking it as in the Trash
	stmt, err := s.db.Prepare("UPDATE document SET in_trash = 1 WHERE id = ?")
	if err != nil {
		return err
	}
	mutex.Lock()
	_, err = stmt.Exec(key)
	mutex.Unlock()
	defer stmt.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) RestoreDocument(key string) error {
	if s.db == nil {
		return errors.New("Cannot restore document-  must open this SQLStore first.")
	}
	// Restoring a Document is just un-marking it as in the Trash
	stmt, err := s.db.Prepare("UPDATE document SET in_trash = 0 WHERE id = ?")
	if err != nil {
		return err
	}
	mutex.Lock()
	_, err = stmt.Exec(key)
	mutex.Unlock()
	defer stmt.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) RenameDocument(key string, newname string) error {
	if s.db == nil {
		return errors.New("Cannot rename document-  must open this SQLStore first.")
	}
	stmt, err := s.db.Prepare("UPDATE document SET name = ?, updated_date = ? WHERE id = ?")
	if err != nil {
		return err
	}
	now := s.timeNow()
	mutex.Lock()
	_, err = stmt.Exec(newname, now, key)
	mutex.Unlock()
	defer stmt.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStore) DuplicateDocument(key string, newname string) (int64, error) {
	if s.db == nil {
		return 0, errors.New("Cannot duplicate document-  must open this SQLStore first.")
	}
	contents, err := s.LoadDocument(key)
	if err != nil {
		return 0, err
	}
	return s.CreateDocument(newname, contents)
}

func (s *SQLStore) LastOpened() (string, error) {
	value, err := s.fetchConfig(LAST_OPENED)
	if err != nil && value == "" {
		value = "0"
	}
	return value, err
}

func (s *SQLStore) fetchConfig(k string) (string, error) {
	if s.db == nil {
		return "", errors.New("Cannot get config value- must open this SQLStore first.")
	}
	var result string
	row := s.db.QueryRow("SELECT value FROM config WHERE key = $1", k)
	err := row.Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		} else {
			return "", err
		}
	}
	return result, nil
}

// TODO: we ought to create a saveConfig(string) method

func (s *SQLStore) timeNow() string { return time.Now().Format("2006-01-02T15:04:05Z") }
