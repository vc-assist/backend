-- name: GetUserFromToken :one
select email from User
inner join (
    select * from ActiveToken where token = ?
) as token on token.userEmail = User.email;

-- name: GetUserFromCode :one
select email from User
inner join (
    select * from VerificationCode where code = ?
) as code on code.userEmail = User.email;

-- name: EnsureUserExists :exec
insert into User(email) values (?)
on conflict do nothing;

-- name: CreateToken :exec
insert into ActiveToken(token, userEmail, expiresAt) values (?, ?, ?);

-- name: CreateVerificationCode :exec
insert into VerificationCode(code, userEmail, expiresAt) values (?, ?, ?);

-- name: DeleteVerificationCode :exec
delete from VerificationCode where code = ?;

-- name: DeleteToken :exec
delete from ActiveToken where token = ?;

