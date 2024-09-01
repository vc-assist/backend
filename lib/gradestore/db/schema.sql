create table UserCourse (
    id integer not null primary key autoincrement,
    user text not null,
    course text not null,
    unique (user, course)
);

create table GradeSnapshot (
    user_course_id integer not null,
    time integer not null,
    value real not null,
    primary key (user_course_id, time),
    foreign key (user_course_id) references UserCourse(id)
        on update cascade
        on delete cascade
);

