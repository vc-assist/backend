-- naming conventions:
-- - "Add" implies an insert with some "on conflict" handling.
-- - "Create" implies an insert with no "on conflict" handling.
-- - "Get" implies a select.
-- - "Delete" implies a delete.

-- *** ACCOUNTS ***

-- name: AddMoodleAccount :one
insert into moodle_account(username, password) values (?, ?)
on conflict do update set
    password = excluded.password
returning id;

-- name: CreateMoodleToken :exec
insert into token(token, moodle_account_id) values (?, ?);

-- name: GetMoodleAccountFromToken :one
select
    moodle_account.id,
    moodle_account.username
from token
inner join moodle_account
    on token.moodle_account_id = moodle_account.id
where token.token = ?;

-- name: GetMoodleAccountFromId :one
select * from moodle_account where id = ?;

-- name: GetAllMoodleAccounts :many
select * from moodle_account;

-- name: GetMoodleUserCount :one
select count(*) from moodle_account;

-- name: AddPSAccount :one
insert into powerschool_account(
    email,
    access_token,
    refresh_token,
    id_token,
    token_type,
    scope,
    expires_at
) values (?, ?, ?, ?, ?, ?, ?)
on conflict do update set
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    id_token = excluded.id_token,
    token_type = excluded.token_type,
    scope = excluded.scope,
    expires_at = excluded.expires_at
returning id;

-- name: CreatePSToken :exec
insert into token(token, powerschool_account_id) values (?, ?);

-- name: GetPSAccountFromToken :one
select 
    powerschool_account.id,
    powerschool_account.email
from token
inner join powerschool_account
    on token.powerschool_account_id = powerschool_account.id
where token.token = ?;

-- name: GetPSUserCount :one
select count(*) from powerschool_account;

-- name: GetPSAccountFromId :one
select * from powerschool_account where id = ?;

-- name: GetAllPSAccounts :many
select * from powerschool_account;



-- *** MOODLE SPECIFIC ***

-- name: DeleteAllMoodleCourses :exec
delete from moodle_course;

-- name: DeleteAllMoodleSections :exec
delete from moodle_section;

-- name: DeleteAllMoodleResources :exec
delete from moodle_resource;

-- name: DeleteAllMoodleChapters :exec
delete from moodle_chapter;

-- name: AddMoodleCourse :exec
insert into moodle_course(id, name) values (?, ?)
on conflict (id) do update
    set name = excluded.name;

-- name: AddMoodleSection :exec
insert into moodle_section(course_id, idx, name) values (?, ?, ?)
on conflict (course_id, idx) do update
    set name = excluded.name;

-- name: AddMoodleResource :exec
insert into moodle_resource(course_id, section_idx, idx, id, type, url, display_content) values (?, ?, ?, ?, ?, ?, ?)
on conflict (course_id, section_idx, idx) do update
    set type = excluded.type,
        url = excluded.url,
        display_content = excluded.display_content;

-- name: AddMoodleChapter :exec
insert into moodle_chapter(course_id, section_idx, resource_idx, id, name, content_html) values (?, ?, ?, ?, ?, ?)
on conflict (id) do update
    set course_id = excluded.course_id,
        section_idx = excluded.section_idx,
        resource_idx = excluded.resource_idx,
        name = excluded.name,
        content_html = excluded.content_html;

-- name: GetMoodleCourses :many
select * from moodle_course where id in (sqlc.slice(ids));

-- name: GetMoodleCourseSections :many
select * from moodle_section where course_id = ?;

-- name: GetMoodleSectionResources :many
select * from moodle_resource where course_id = ? and section_idx = ?;

-- name: GetMoodleResourceChapters :many
select name, id from moodle_chapter where
    course_id = ? and
    section_idx = ? and
    resource_idx = ?;

-- name: GetMoodleChapterContent :one
select content_html from moodle_chapter where id = ?;

-- name: GetMoodleFileResource :one
select url from moodle_resource where id = ? and type = 1;

-- name: GetAllMoodleCourses :many
select * from moodle_course;

-- name: AddMoodleUserCourse :exec
insert into moodle_user_course(account_id, course_id) values (?, ?)
on conflict do nothing;

-- name: GetMoodleUserCourses :many
select moodle_user_course.course_id from moodle_user_course where moodle_user_course.account_id = ?;



-- *** POWERSCHOOL SPECIFIC ***

-- name: AddPSCachedData :exec
insert into powerschool_data_cache(account_id, data) values (?, ?)
on conflict do update set
    data = excluded.data;

-- name: GetPSCachedData :one
select data from powerschool_data_cache where account_id = ?;



-- *** WEIGHTS SPECIFIC ***

-- name: AddWeightCourse :one
insert into weight_course(actual_course_id, actual_course_name) values (?, ?)
on conflict do update
    set actual_course_name = excluded.actual_course_name
returning id;

-- name: AddWeightCategory :exec
insert into weight_category(weight_course_id, category_name, weight) values (?, ?, ?);

-- name: GetWeightCourseCategories :many
select category_name, weight from weight_category
inner join weight_course on weight_course.id = weight_category.weight_course_id
where weight_course.actual_course_id = ?;



-- *** SNAPSHOT SPECIFIC ***

-- name: CreateSnapshotSeries :one
insert into gradesnapshot_series(powerschool_account_id, course_id, start_time) values (?, ?, ?)
returning id;

-- name: CreateSnapshot :exec
insert into gradesnapshot(series_id, value) values (?, ?);

-- name: GetSnapshotSeries :many
select id, course_id, start_time from gradesnapshot_series
where powerschool_account_id = ? and course_id = ?
order by start_time asc;

-- name: GetLatestSnapshotSeries :one
select id, course_id, start_time from gradesnapshot_series
where powerschool_account_id = ? and course_id = ?
order by start_time desc 
limit 1;

-- name: GetSnapshotSeriesSnapshots :many
select value from gradesnapshot
where series_id = ?
order by rowid asc;

-- name: GetSnapshotSeriesCount :one
select count(gradesnapshot.value) from gradesnapshot_series
inner join gradesnapshot on gradesnapshot_series.id = gradesnapshot.series_id
where gradesnapshot_series.id = ?;

-- name: GetSnapshotSeriesCourseIds :many
select distinct course_id from gradesnapshot_series
where powerschool_account_id = ?;

