-- name: GetExplicitLinks :many
select * from ExplicitLink
where (leftSet = ?1 and rightSet = ?2) or
    (rightSet = ?2 and leftSet = ?1);

-- name: CreateExplicitLink :exec
insert into ExplicitLink(leftSet, leftKey, rightSet, rightKey) values (?, ?, ?, ?);

-- name: DeleteExplicitLink :exec
delete from ExplicitLink where
    leftSet = ? and
    leftKey = ? and
    rightSet = ? and
    rightKey = ?;

-- name: CreateKnownSet :exec
insert into KnownSet(setname) values (?) on conflict (setname) do nothing;

-- name: CreateKnownKey :exec
insert into KnownKey(setname, value, lastSeen) values (?, ?, ?)
on conflict (setname, value) do update set
    lastSeen = EXCLUDED.lastSeen;

-- name: GetKnownSets :many
select * from KnownSet;

-- name: GetKnownKeys :many
select * from KnownKey where setname = ?;

-- name: GetKnownKeyBefore :many
select * from KnownKey where lastSeen < ?;

-- name: DeleteKnownSets :exec
delete from KnownSet where setname in (sqlc.slice(sets));

-- name: DeleteKeysBefore :exec
delete from KnownKey where setname = ? and lastSeen < ?;

