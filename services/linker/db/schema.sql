create table ExplicitLink (
    leftSet text not null,
    leftKey text not null,
    rightSet text not null,
    rightKey text not null,
    primary key (leftSet, leftKey, rightSet, rightKey)
);

create table KnownSet (
    set text not null primary key
);

create table KnownKey (
    set text not null,
    value text not null,
    lastSeen integer not null,
    primary key (set, value)
);

