create table StudentData (
    studentId text not null primary key,
    cached blob not null,
    createdAt integer not null
);

create table KnownCourse (
    guid text not null primary key,
    name text not null,
    period text,
    teacherFirstName text,
    teacherLastName text,
    teacherEmail text,
    room text
);

