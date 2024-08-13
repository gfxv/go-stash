package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"strings"
)

const DB_PATH = "meta.db"
const DB_DRIVER = "sqlite3"

type DB struct {
	database *sql.DB
}

func NewDB(root string) (*DB, error) {
	//fullPath := fmt.Sprintf("%s/%s", root, DB_PATH)
	fullPath := filepath.Join(root, DB_PATH)
	database, err := sql.Open(DB_DRIVER, fullPath)
	if err != nil {
		return nil, err
	}
	db := &DB{
		database: database,
	}
	err = db.init()
	return db, err
}

func (db *DB) init() (err error) {
	stmt, err := db.database.Prepare("create table if not exists keys (" +
		"id integer primary key autoincrement," +
		"key text not null," +
		"hash text not null unique" +
		")",
	)
	defer func(stmt *sql.Stmt) {
		if tmpErr := stmt.Close(); tmpErr != nil {
			fmt.Println(err)
			err = tmpErr
		}
	}(stmt)
	_, err = stmt.Exec()
	return
}

// Add adds key-hash records to database
func (db *DB) Add(key string, hashes []string) error {
	stmtStr := "insert into keys (key, hash) values"
	var vals []interface{}
	for _, h := range hashes {
		stmtStr += " (?, ?),"
		vals = append(vals, key, h)
	}
	stmtStr = strings.TrimSuffix(stmtStr, ",")
	stmt, err := db.database.Prepare(stmtStr)
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		if tmpErr := stmt.Close(); tmpErr != nil {
			err = tmpErr
		}
	}(stmt)

	fmt.Println(stmtStr)
	_, err = stmt.Exec(vals...)

	return err
}

// GetByKey returns hashes associated with given key
func (db *DB) GetByKey(key string) ([]string, error) {
	rows, err := db.database.Query("select hash from keys where key = ?", key)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		if tmpErr := rows.Close(); tmpErr != nil {
			err = tmpErr
		}
	}(rows)

	hashes := make([]string, 0)
	for rows.Next() {
		var hash string
		err = rows.Scan(&hash)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	return hashes, err
}

// Remove TODO
func (db *DB) Remove(key string) error { return nil }
