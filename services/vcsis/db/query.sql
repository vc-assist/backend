-- name: CacheStudentData :exec
insert into StudentData(student_id, data, last_updated) values (?, ?, ?)
on conflict do update
    set data = excluded.data,
        last_updated = excluded.last_updated;

-- name: GetStudentData :one
select data, last_updated from StudentData
where student_id = ?;

-- name: GetAllStudents :many
select student_id from StudentData;

