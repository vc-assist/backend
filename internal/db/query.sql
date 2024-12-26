-- *** GENERIC ***

-- name: SetMoodleAccount :one
insert into moodle_account(username) values (?)
-- technically this is a useless update, but on "conflict do nothing" will not
-- return anything when a conflict is encountered so on "conflict do update" is
-- needed to have the updated/inserted row's id returned
on conflict do update set username = excluded.username
returning id;

-- name: SetPSAccount :one
insert into powerschool_account(email) values (?)
on conflict do update set email = excluded.email
returning id;

-- name: CreateMoodleToken :exec
insert into token(token, moodle_account_id) values (?, ?);

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

-- name: SetPSCachedData :exec
insert into powerschool_data_cache(account_id, data) values (?, ?)
on conflict do update set
    data = excluded.data;

-- name: GetPSCachedData :one
select data from powerschool_data_cache where account_id = ?;

