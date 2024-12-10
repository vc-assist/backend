-- name: GetOAuthBefore :many
select * from OAuth where expires_at < ?;

-- name: DeleteOAuthBefore :exec
delete from OAuth where expires_at < ?;

-- name: GetUsernamePassword :one
select username, password from UsernamePassword where
namespace = ? and id = ?;

-- name: GetOAuth :one
select token, refresh_url, client_id, expires_at from OAuth where
namespace = ? and id = ?;

-- name: CreateOAuth :exec
insert into OAuth(namespace, id, token, refresh_url, client_id, expires_at) values (?, ?, ?, ?, ?, ?)
on conflict do update set
    token = EXCLUDED.token,
    refresh_url = EXCLUDED.refresh_url,
    client_id = EXCLUDED.client_id,
    expires_at = EXCLUDED.expires_at;

-- name: CreateUsernamePassword :exec
insert into UsernamePassword(namespace, id, username, password) values (?, ?, ?, ?)
on conflict do update set
    username = EXCLUDED.username,
    password = EXCLUDED.password;

-- name: FindIdFromOAuthToken :one 
select id from OAuth where token = ?;

-- name: FindIdFromUsername :one 
select id from UsernamePassword where username = ?;

-- name: CreateSessionToken :exec
insert into SessionToken(token, OAuthId, UsernamePasswordId) values (?, ?, ?)
on conflict do nothing;

-- name: FindSessionToken :one
select * from SessionToken where token = ?;
