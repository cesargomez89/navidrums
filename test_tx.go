package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type dbOps interface {
	Get(dest interface{}, query string, args ...interface{}) error
}

type DB struct {
	dbOps
	root *sqlx.DB
}

func (db *DB) RunInTx(ctx context.Context, fn func(txDB *DB) error) error {
	tx, err := db.root.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	txDB := &DB{
		dbOps: tx,
		root:  db.root,
	}

	if err := fn(txDB); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) TrackCount() (int, error) {
	var count int
	err := db.Get(&count, "SELECT 1")
	return count, err
}

func main() {
	realDB, _ := sqlx.Open("sqlite", ":memory:")
	db := &DB{dbOps: realDB, root: realDB}

	err := db.RunInTx(context.Background(), func(txDB *DB) error {
		c, err := txDB.TrackCount()
		fmt.Println("Count inside TX:", c)
		return err
	})
	fmt.Println("Err:", err)
}
