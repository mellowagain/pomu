create table users
(
    id     varchar      not null primary key,
    name   varchar(128) not null,
    avatar varchar      not null
);

create table videos
(
    id         varchar                  not null primary key,
    submitters character varying[]      not null,
    start      timestamp with time zone not null
);
