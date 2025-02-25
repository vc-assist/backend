-- *** ACCOUNTS ***

-- stores a certain user's moodle account
create table moodle_account (
    id integer not null primary key autoincrement,
    -- this should be lowercase, space trimmed, without the @warriorlife.net suffix
    username text not null unique,
    password text not null
);

-- stores a certain user's powerschool account/token
create table powerschool_account (
    id integer not null primary key autoincrement,
    -- this should be lowercase and space trimmed
    email text not null unique,
    access_token text not null,
    refresh_token text not null,
    id_token text not null,
    token_type text not null,
    scope text not null,
    expires_at timestamp not null
);

-- stores a VC Assist authentication token that can refer to a moodle credential or powerschool credential
create table token (
    token text not null primary key,
    -- only one of these should be defined
    moodle_account_id integer,
    powerschool_account_id integer,
    foreign key (moodle_account_id) references moodle_account(id)
        on update cascade
        on delete cascade,
    foreign key (powerschool_account_id) references powerschool_account(id)
        on update cascade
        on delete cascade
);



-- *** MOODLE SPECIFIC ***

create table moodle_course (
    id integer not null primary key,
    name text not null
);

create table moodle_section (
    course_id integer not null,
    idx integer not null,
    name text not null,

    primary key (course_id, idx),
    foreign key (course_id) references moodle_course(id)
        on update cascade
        on delete cascade
);

create table moodle_resource (
    course_id integer not null,
    section_idx integer not null,
    -- this is the index of the resource in its
    -- containing section, ex.
    -- Lesson Plans (section, idx 2):
    -- - Quarter 1 (Book, idx 0)
    -- - Quarter 2 (Book, idx 1)
    -- - Some URL (URL, idx 2)
    idx integer not null,
    -- this may or may not also have an actual id depending on what is stored
    id integer,
    -- 0: generic url
    -- 1: file
    -- 2: book
    -- 3: html area
    type integer not null,
    url text not null,
    -- this will be the name for urls/books
    -- this will be the html for html areas
    display_content text not null,

    primary key (course_id, section_idx, idx),
    foreign key (course_id, section_idx) references moodle_section(course_id, idx)
        on update cascade
        on delete cascade,
    foreign key (course_id) references moodle_course(id)
        on update cascade
        on delete cascade
);

create table moodle_chapter (
    course_id integer not null,
    section_idx integer not null,
    resource_idx integer not null,

    id integer not null primary key,
    name text not null,
    content_html text not null,

    foreign key (course_id, section_idx, resource_idx) references moodle_resource(course_id, section_idx, idx)
        on update cascade
        on delete cascade
);

create table moodle_user_course (
    account_id integer not null,
    course_id integer not null,
    foreign key (account_id) references moodle_account(id)
        on update cascade
        on delete cascade,
    foreign key (course_id) references moodle_course(id)
        on update cascade
        on delete cascade,
    unique (account_id, course_id)
);



-- *** POWERSCHOOL SPECIFIC ***

create table powerschool_data_cache (
    account_id integer not null primary key,
    data blob not null,
    foreign key (account_id) references powerschool_account(id)
        on update cascade
        on delete cascade
);



-- *** WEIGHTS SPECIFIC ***

create table weight_course (
    id integer not null primary key autoincrement,
    actual_course_id text not null unique,
    actual_course_name text not null
);

create table weight_category (
    weight_course_id integer not null,
    category_name text not null,
    -- weight is a float from 0-1
    weight real not null,
    primary key (weight_course_id, category_name),
    foreign key (weight_course_id) references weight_course(id)
        on update cascade
        on delete cascade
);



-- *** SNAPSHOT SPECIFIC ***

-- starts a series on a certain date
-- here's an example of the relationship of this to gradesnapshot
-- 
-- gradesnapshot_series(id = 0, powerschool/course_id = ..., start_time = 12/20/2020)
--   * gradesnapshot(series_id = 0, value = 90) <- date: 12/20/2020
--   * gradesnapshot(series_id = 0, value = 90) <- date: 12/21/2020
--   * gradesnapshot(series_id = 0, value = 95) <- date: 12/22/2020
--   * gradesnapshot(series_id = 0, value = 92) <- date: 12/23/2020
--
-- why: this is because storing time series data tends to take up a lot of space, so we'll
-- attempt to reduce the number of fields duplicated over the time dimension
-- (imagine storing powerschool/course_id for every snapshot row, that's a lot of wasted space)
create table gradesnapshot_series (
    id integer not null primary key autoincrement,
    powerschool_account_id integer not null,
    course_id text not null,
    start_time datetime not null,
    unique (powerschool_account_id, course_id, start_time),
    foreign key (powerschool_account_id) references powerschool_account(id)
        on update cascade
        on delete cascade
);

-- stores a single grade snapshot, see gradesnapshot_series for more information
-- 
-- note: the "id" for this row is just the built in sqlite "rowid" field which can be used
-- with "order by" in a select to ensure that the snapshots are queried in order
create table gradesnapshot (
    series_id integer not null,
    value real not null,
    foreign key (series_id) references gradesnapshot_series(id)
        on update cascade
        on delete cascade
);

