create table Student (
    id text not null primary key
);

create table OAuthToken (
    studentId text not null primary key,
    token text not null,
    expiresAt integer not null,
    foreign key (studentId) references Student(id)
        on update cascade
        on delete cascade
);

create table StudentData (
    studentId text not null primary key,
    cached blob not null,
    createdAt integer not null,
    foreign key (studentId) references Student(id)
        on update cascade
        on delete cascade
);

