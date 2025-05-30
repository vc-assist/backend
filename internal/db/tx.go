package db

import (
	"database/sql"
)

// MakeTx is a function that creates a db transaction
type MakeTx = func() (tx *Queries, discard, commit func() error, err error)

func NewMakeTx(dbtx *sql.DB) MakeTx {
	return func() (tx *Queries, discard, commit func() error, err error) {
		sqltx, err := dbtx.Begin()
		if err != nil {
			return nil, nil, nil, err
		}
		txqry := New(sqltx)
		return txqry,
			func() error {
				return sqltx.Rollback()
			},
			func() error {
				return sqltx.Commit()
			},
			nil
	}
}
