-- name: DeleteAllCourses :exec
delete from Course;

-- name: DeleteAllSections :exec
delete from Section;

-- name: DeleteAllResources :exec
delete from Resource;

-- name: DeleteAllChapters :exec
delete from Chapter;

-- name: NoteCourse :exec
insert into Course(id, name) values (?, ?)
on conflict (id) do update
    set name = excluded.name;

-- name: NoteSection :exec
insert into Section(course_id, idx, name) values (?, ?, ?)
on conflict (course_id, idx) do update
    set name = excluded.name;

-- name: NoteResource :exec
insert into Resource(course_id, section_idx, idx, type, url, display_content) values (?, ?, ?, ?, ?, ?)
on conflict (course_id, section_idx, idx) do update
    set type = excluded.type,
        url = excluded.url,
        display_content = excluded.display_content;

-- name: NoteChapter :exec
insert into Chapter(course_id, section_idx, resource_idx, id, name, content_html) values (?, ?, ?, ?, ?, ?)
on conflict (id) do update
    set course_id = excluded.course_id,
        section_idx = excluded.section_idx,
        resource_idx = excluded.resource_idx,
        name = excluded.name,
        content_html = excluded.content_html;

-- name: GetCourses :many
select * from Course where id in (sqlc.slice(ids));

-- name: GetCourseSections :many
select * from Section where course_id = ?;

-- name: GetSectionResources :many
select * from Resource where course_id = ? and section_idx = ?;

-- name: GetResourceChapters :many
select * from Chapter where
    course_id = ? and
    section_idx = ? and
    resource_idx = ?;

-- name: GetChapterContent :one
select content_html from Chapter where id = ?;

