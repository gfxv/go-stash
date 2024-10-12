package cas

import (
	"database/sql"
	"errors"
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

// NewDB creates a new instance of DB and initializes the database connection.
//
// This method takes a root directory path, constructs the full path to the
// database file, and attempts to open a connection to the database.
// If the connection is successful, it initializes the database
// and returns a pointer to the DB instance. In case of any errors during these
// processes, an error is returned.
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

// Add inserts key-hash records into the database.
//
// This method takes a key and a slice of hash strings and adds them to the
// `keys` table in the database. It validates that the key and hashes are
// not empty and constructs an SQL insert statement for the operation. If
// any of the inputs are invalid or if an error occurs during the database
// operations, an error is returned.
func (db *DB) Add(key string, hashes []string) error {
	const op = "cas.db.Add"

	if len(key) == 0 {
		return fmt.Errorf("%s: %w", op, errors.New("empty key"))
	}

	if len(hashes) == 0 {
		return fmt.Errorf("%s: %w", op, errors.New("empty hash list"))
	}

	stmtStr := "insert into keys (key, hash) values"
	var vals []interface{}
	for _, h := range hashes {
		if len(h) == 0 {
			return fmt.Errorf("%s: %w", op, errors.New("empty hash"))
		}
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

// GetByKey retrieves hashes associated with a given key from the database.
//
// This method takes a key as input and queries the `keys` table to retrieve
// all associated hash values. If the query is successful, it returns a slice
// of strings containing the hashes. If an error occurs during the query or
// while scanning the results, an error is returned.
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

// GetKeysByChunks retrieves a chunk of distinct keys from the database.
//
// This method takes an offset as input and queries the `keys` table to fetch
// a limited number of distinct keys based on the specified offset.
// Chunk size is defined by DB_CHUNK_SIZE value.
// If the query is successful, it returns a slice of strings containing the keys.
// If an error occurs during the query or while scanning the results, an error is returned.
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

// RemoveByKey deletes all records associated with a given key from the database.
//
// This method takes a key as input and executes a delete operation on the
// `keys` table, removing all entries that match the specified key. If an
// error occurs during the preparation or execution of the SQL statement,
// an error is returned.
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
