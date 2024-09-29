package cas

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"strings"
)

const DB_PATH = "meta.db"
const DB_DRIVER = "sqlite3"

// TODO: move it to config file
const DB_CHUNK_SIZE = 100

type DB struct {
	database *sql.DB
}

func NewDB(root string) (*DB, error) {
	const op = "cas.db.NewDB"

	fullPath := filepath.Join(root, DB_PATH)
	database, err := sql.Open(DB_DRIVER, fullPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	db := &DB{
		database: database,
	}
	err = db.init()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return db, err
}

func (db *DB) init() (err error) {
	const op = "cas.db.init"

	stmt, err := db.database.Prepare("create table if not exists keys (" +
		"id integer primary key autoincrement," +
		"key text not null," +
		"hash text not null unique" +
		")",
	)
	defer func(stmt *sql.Stmt) {
		if tmpErr := stmt.Close(); tmpErr != nil {
			err = fmt.Errorf("%s: %w", op, tmpErr)
		}
	}(stmt)
	_, err = stmt.Exec()
	if err != nil {
		err = fmt.Errorf("%s: %w", op, err)
	}
	return
}

// Add adds key-hash records to database
func (db *DB) Add(key string, hashes []string) error {
	const op = "cas.db.Add"

	stmtStr := "insert into keys (key, hash) values"
	var vals []interface{}
	for _, h := range hashes {
		stmtStr += " (?, ?),"
		vals = append(vals, key, h)
	}
	stmtStr = strings.TrimSuffix(stmtStr, ",")
	stmt, err := db.database.Prepare(stmtStr)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func(stmt *sql.Stmt) {
		if tmpErr := stmt.Close(); tmpErr != nil {
			err = tmpErr
		}
	}(stmt)

	_, err = stmt.Exec(vals...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetByKey returns hashes associated with given key
func (db *DB) GetByKey(key string) ([]string, error) {
	const op = "cas.db.GetByKey"

	rows, err := db.database.Query("select hash from keys where key = ?", key)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func(rows *sql.Rows) {
		if tmpErr := rows.Close(); tmpErr != nil {
			err = fmt.Errorf("%s: %w", op, tmpErr)
		}
	}(rows)

	hashes := make([]string, 0)
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		hashes = append(hashes, hash)
	}

	return hashes, err
}

func (db *DB) GetKeysByChunks(offset int) ([]string, error) {
	const op = "cas.db.GetKeysByChunks"

	rows, err := db.database.Query("select distinct key from keys limit ? offset ?", DB_CHUNK_SIZE, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// RemoveByKey ...
func (db *DB) RemoveByKey(key string) error {
	const op = "cas.db.Remove"

	stmt, err := db.database.Prepare("delete from keys where key = ?")
	defer func(stmt *sql.Stmt) {
		if tmpErr := stmt.Close(); tmpErr != nil {
			err = fmt.Errorf("%s: %w", op, tmpErr)
		}
	}(stmt)

	_, err = stmt.Exec(key)
	if err != nil {
		err = fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
