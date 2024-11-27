--all snake case

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

-- name: GerParentFromToken :one 
select email from Parent
inner join (
    select * from ParentToken where token = ?
) as token on token.parentEmail = Parent.email;

-- name: GetParentFromCode :one
select email from Parent
inner join (
    select * from ParentVerificationCode where code = ?
) as code on code.ParentVerificationCode = Parent.email;

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

-- name: CreateParent :exec
insert into Parent(email, userEmail) values(?, ?)
on conflict do nothing;

-- name: CreateParentToken :exec
insert into ParentToken(token, parentEmail, expiresAt) values (?, ?, ?);

-- name: CreateParentVerificationCode :exec
insert into ParentVerificationCode(code, parentEmail, expiresAt) values (?, ?, ?);

-- name: DeleteParentVerificationCode :exec
delete from ParentVerificationCode where code = ?;

-- name: DeleteParentToken :exec
delete from ParentToken where token = ?;

-- name: CheckParent :one
SELECT UserEmail from Parent where email = ?;

-- name: CheckParentVerification :one
SELECT count(*) from ParentVerificationCode where expiresAt > ? and code = ? and parentEmail = ?;