-- name: GetCachedStudentData :one
select expiresAt, cached from StudentDataCache where studentId = ?;

-- name: SetCachedStudentData :exec
insert into StudentDataCache(studentId, cached, expiresAt) values (?, ?, ?)
on conflict do update set
    cached = EXCLUDED.cached,
    expiresAt = EXCLUDED.expiresAt;

-- name: DeleteCachedStudentDataBefore :exec
delete from StudentDataCache where expiresAt < sqlc.arg(before);

-- name: GetStudents :many
select studentId from StudentDataCache;

