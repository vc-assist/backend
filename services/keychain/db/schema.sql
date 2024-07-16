create table OAuth (
    namespace text not null,
    id text not null,
    token text not null,
    refresh_url text not null,
    client_id text not null,
    expires_at integer not null,
    primary key (namespace, id)
);

create table UsernamePassword (
    namespace text not null,
    id text not null,
    username text not null,
    password text not null,
    primary key (namespace, id)
);

