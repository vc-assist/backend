create table UserCourse (
    id integer not null primary key autoincrement,
    user text not null,
    course text not null,
    unique (user, course)
);

create table GradeSnapshot (
    userCourseId integer not null,
    time integer not null,
    value real not null,
    primary key (userCourseId, time),
    foreign key (userCourseId) references UserCourse(id)
        on update cascade
        on delete cascade
);

