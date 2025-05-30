// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: query.sql

package db

import (
	"context"
	"time"
)

const cacheStudentData = `-- name: CacheStudentData :exec
insert into StudentData(student_id, data, last_updated) values (?, ?, ?)
on conflict do update
    set data = excluded.data,
        last_updated = excluded.last_updated
`

type CacheStudentDataParams struct {
	StudentID   string
	Data        []byte
	LastUpdated time.Time
}

func (q *Queries) CacheStudentData(ctx context.Context, arg CacheStudentDataParams) error {
	_, err := q.db.ExecContext(ctx, cacheStudentData, arg.StudentID, arg.Data, arg.LastUpdated)
	return err
}

const getAllStudents = `-- name: GetAllStudents :many
select student_id from StudentData
`

func (q *Queries) GetAllStudents(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getAllStudents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var student_id string
		if err := rows.Scan(&student_id); err != nil {
			return nil, err
		}
		items = append(items, student_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStudentData = `-- name: GetStudentData :one
select data, last_updated from StudentData
where student_id = ?
`

type GetStudentDataRow struct {
	Data        []byte
	LastUpdated time.Time
}

func (q *Queries) GetStudentData(ctx context.Context, studentID string) (GetStudentDataRow, error) {
	row := q.db.QueryRowContext(ctx, getStudentData, studentID)
	var i GetStudentDataRow
	err := row.Scan(&i.Data, &i.LastUpdated)
	return i, err
}
