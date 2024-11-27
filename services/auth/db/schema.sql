create table User (
    email text not null primary key 

);
create table Parent (
    email text not null primary key, 
    userEmail text not null,
    foreign key (userEmail) references User(email)
);
--these are user specific tokens 
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
-- these are parent specific tokens
create table ParentToken (
    token text not null primary key,
    parentEmail text not null,
    expiresAt int not null,
    foreign key (parentEmail) references Parent(parentEmail)
);
create table ParentVerificationCode (
    code text not null primary key,
    parentEmail text not null,
    expiresAt int not null,
    foreign key (parentEmail) references Parent(parentEmail)
);

