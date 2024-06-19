create table Student (
    id text not null primary key,
    token text not null
);

create table AllStudents (
    studentId text not null primary key,
    cached blob not null,
    foreign key (studentId) references Student(id)
        on update cascade
        on delete cascade
);

create table CourseMeetingList (
    studentId text not null primary key,
    cached blob not null,
    foreign key (studentId) references Student(id)
        on update cascade
        on delete cascade
);

create table StudentData (
    studentId text not null primary key,
    cached blob not null,
    foreign key (studentId) references Student(id)
        on update cascade
        on delete cascade
);

