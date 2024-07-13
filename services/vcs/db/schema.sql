create table StudentDataCache (
    studentId text not null primary key,
    cached blob not null,
    expiresAt integer not null
);
