package store

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const DBVersion int = 6

type SQLiteStorage struct {
	db          *sql.DB
	writeDBLock sync.Mutex
	dbWg        sync.WaitGroup
}

func NewSQLiteStorage(databasePath, createTableScript string) (*SQLiteStorage, error) {
	self := new(SQLiteStorage)
	i := strings.LastIndex(databasePath, ".")
	if i < 0 {
		return nil, errors.New(`new sqlite database error: can't find "."`)
	}
	statePath := string(databasePath[0:i]) + "-state.db"
	//state
	connState, err := sql.Open("sqlite3", statePath)
	if err != nil {
		return nil, err
	}
	_, err = connState.Exec("PRAGMA synchronous = NORMAL")
	if err != nil {
		return nil, err
	}
	_, err = connState.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	self.db = connState

	createStateScript := GetCreateTables()
	_, err = self.db.Exec(createStateScript)
	if err != nil {
		return nil, err
	}

	if len(createTableScript) > 0 {
		_, err = self.db.Exec(createTableScript)
		if err != nil {
			return nil, err
		}
	}

	return self, nil
}

func (this *SQLiteStorage) Exec(query string, args ...interface{}) (bool, error) {
	stmtState, err := this.db.Prepare(query)
	if err != nil {
		return false, err
	}
	_, err = stmtState.Exec(args...)
	if err != nil {
		return false, err
	}
	stmtState.Close()
	return true, nil
}

func (this *SQLiteStorage) Query(query string, args ...interface{}) (*sql.Rows, error) {
	stmtState, err := this.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	rows, err := stmtState.Query(args...)
	stmtState.Close()
	return rows, err
}

func (this *SQLiteStorage) Close() error {
	return this.db.Close()
}
