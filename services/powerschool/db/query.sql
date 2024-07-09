-- name: GetOAuthToken :one
select * from OAuthToken
where studentId = ? limit 1;

-- name: GetExpiredTokens :many
select * from OAuthToken
where expiresAt < ?;

-- name: DeleteExpiredOAuthTokens :exec
delete from OAuthToken
where expiresAt < ?;

-- name: CreateStudent :exec
insert into Student(id)
values (?)
on conflict (id) do nothing;

-- name: CreateOrUpdateOAuthToken :exec
insert into OAuthToken(studentId, token, expiresAt)
values (?, ?, ?)
on conflict (studentId) do update set
    token = EXCLUDED.token,
    expiresAt = EXCLUDED.expiresAt;

-- name: CreateOrUpdateStudentData :exec
insert into StudentData(studentId, cached, createdAt)
values (?, ?, ?)
on conflict (studentId)
    do update set
        cached = EXCLUDED.cached,
        createdAt = EXCLUDED.createdAt;

-- name: CreateOrUpdateKnownCourse :exec
insert into KnownCourse(guid, name, period, teacherFirstName, teacherLastName, teacherEmail, room)
values (?, ?, ?, ?, ?, ?, ?)
on conflict (guid)
    do update set
        guid = EXCLUDED.guid,
        name = EXCLUDED.name,
        period = EXCLUDED.period,
        teacherFirstName = EXCLUDED.teacherFirstName,
        teacherLastName = EXCLUDED.teacherLastName,
        teacherEmail = EXCLUDED.teacherEmail,
        room = EXCLUDED.room;

-- name: GetKnownCourses :many
select * from KnownCourse;

