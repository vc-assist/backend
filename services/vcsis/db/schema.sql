create table StudentData (
    student_id text not null primary key,
    data blob not null,
    last_updated datetime not null
);
