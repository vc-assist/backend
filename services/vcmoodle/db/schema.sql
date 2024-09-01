create table Course (
    id integer not null primary key,
    name text not null
);

create table Section (
    course_id integer not null,
    idx integer not null,
    name text not null,

    primary key (course_id, idx),
    foreign key (course_id) references Course(id)
);

create table Resource (
    course_id integer not null,
    section_idx integer not null,
    -- this is the index of the resource in its
    -- containing section, ex.
    -- Lesson Plans (Section, idx 2):
    -- - Quarter 1 (Book, idx 0)
    -- - Quarter 2 (Book, idx 1)
    -- - Some URL (URL, idx 2)
    idx integer not null,
    -- 0: generic url
    -- 1: book
    -- 2: html area
    type integer not null,
    url text not null,
    -- this will be the name for urls/books
    -- this will be the html for html areas
    display_content text not null,

    primary key (course_id, section_idx, idx),
    foreign key (course_id, section_idx) references Section(course_id, idx),
    foreign key (course_id) references Course(id)
);

create table Chapter (
    course_id integer not null,
    section_idx integer not null,
    resource_idx integer not null,

    id integer not null primary key,
    name text not null,
    content_html text not null,

    foreign key (course_id, section_idx, resource_idx) references Resource(course_id, section_idx, idx)
);

