-- name: CreateStudent :exec
insert into Student(id, username, password) values (?, ?, ?)
on conflict (id) do update set
    username = EXCLUDED.username,
    password = EXCLUDED.password;

-- name: GetStudent :one
select * from Student where id = ?;
