-- name: GetGradeSnapshots :many
select FoundCourses.course, time, value from GradeSnapshot
inner join (
    select * from UserCourse where user = ?
) as FoundCourses
    on FoundCourses.id = userCourseId
order by (FoundCourses.course, time);

-- name: CreateUserCourse :one
insert into UserCourse(user, course)
values (?, ?)
on conflict do nothing
returning id;

-- name: CreateGradeSnapshot :exec
insert into GradeSnapshot(userCourseId, time, value)
values (?, ?, ?);

