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

