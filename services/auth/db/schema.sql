create table User (
    email text not null primary key
);

create table ActiveToken (
    token text not null primary key,
    userEmail text not null,
    expiresAt int not null,
    foreign key (userEmail) references User(email)
);

create table VerificationCode (
    code text not null primary key,
    userEmail text not null,
    expiresAt int not null,
    foreign key (userEmail) references User(email)
);

