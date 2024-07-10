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
insert into KnownSet(set) values (?) on conflict do nothing;

-- name: CreateKnownKey :exec
insert into KnownKey(set, value, lastSeen) values (?, ?, ?);

-- name: GetKnownSets :many
select * from KnownSet;

-- name: GetKnownKeys :many
select * from KnownKey where set = ?;

-- name: GetKnownKeyBefore :many
select * from KnownKey where lastSeen < ?;

