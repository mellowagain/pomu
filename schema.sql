create table users
(
    id     varchar      not null primary key,
    name   varchar(128) not null,
    avatar varchar      not null
);

create table videos
(
    id           varchar                  not null primary key,
    submitters   character varying[]      not null,
    start        timestamp with time zone not null,
    finished     boolean default false    not null,
    title        varchar                  not null,
    channel_name varchar                  not null,
    channel_id   varchar                  not null,
    thumbnail    varchar                  not null,
    file_size    integer default 0        not null,
    video_length integer default 0        not null
);

comment on column videos.file_size is 'in bytes';
comment on column videos.video_length is 'in seconds';
