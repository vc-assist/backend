create table OAuth (
    namespace text not null,
    id int primary key autoincrement,
    token text not null,
    email text not null,
    refresh_url text not null,
    client_id text not null,
    expires_at integer not null,
    primary key (namespace, id)
);

create table UsernamePassword (
    namespace text not null,
    id int primary key autoincrement,
    username text not null,
    password text not null,
    primary key (namespace, id)
);
-- this token is for moodle and powerschool combined so users dont have to login everytime
-- add documentation later  Author Justin Shi
CREATE TABLE SessionToken (
    token text not null primary key,
    oAuthId int,
    usernamePasswordId int,
    FOREIGN KEY (oAuthId) REFERENCES OAuth(id),
    FOREIGN KEY (usernamePasswordId) REFERENCES UsernamePassword(id)
);

