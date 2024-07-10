create table ExplicitLink (
    leftSet text not null,
    leftKey text not null,
    rightSet text not null,
    rightKey text not null,
    primary key (leftSet, leftKey, rightSet, rightKey)
);

create table KnownSet (
    -- "setname" instead of "set" used because "set" is a reserved keyword
    setname text not null primary key
);

create table KnownKey (
    setname text not null,
    value text not null,
    lastSeen integer not null,
    primary key (setname, value)
);

