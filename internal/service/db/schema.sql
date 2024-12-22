-- stores a certain user's moodle account
create table moodle_account (
    id integer not null primary key autoincrement,
    -- this should be lowercase, space trimmed, without the @warriorlife.net suffix
    username text not null unique
);

-- stores a certain user's powerschool account/token
create table powerschool_account (
    id integer not null primary key autoincrement,
    -- this should be lowercase and space trimmed
    email text not null unique
);

-- stores a VC Assist authentication token that can refer to a moodle credential or powerschool credential
create table token (
    token text not null primary key,
    -- only one of these should be defined
    moodle_account_id integer,
    powerschool_account_id integer,
    foreign key (moodle_account_id) references moodle_account(id),
    foreign key (powerschool_account_id) references powerschool_account(id)
);

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
);

-- stores a single grade snapshot, see gradesnapshot_series for more information
create table gradesnapshot (
    series_id integer not null,
    value real not null,
    foreign key (series_id) references gradesnapshot_series
);

