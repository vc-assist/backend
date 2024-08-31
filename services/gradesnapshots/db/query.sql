-- name: GetGradeSnapshots :many
select FoundCourses.course, json_group_array(json_array(time, value)) as grades from GradeSnapshot
inner join (
    select * from UserCourse where user = ?
) as FoundCourses
    on FoundCourses.id = user_course_id
group by FoundCourses.course;

-- name: CreateUserCourse :exec
insert into UserCourse(user, course)
values (?, ?)
on conflict do nothing;

-- name: GetUserCourseId :one
select id from UserCourse where user = ? and course = ?;

-- name: CreateGradeSnapshot :exec
insert into GradeSnapshot(user_course_id, time, value)
values (?, ?, ?);

-- name: DeleteGradeSnapshotsIn :exec
delete from GradeSnapshot where
rowid in (
    select rowid from GradeSnapshot as SubSnapshot
    inner join (
        select * from UserCourse where user = ?
    ) as FoundCourses
        on FoundCourses.id = SubSnapshot.user_course_id
    and SubSnapshot.time > sqlc.arg(after)
    and SubSnapshot.time < sqlc.arg(before)
)

