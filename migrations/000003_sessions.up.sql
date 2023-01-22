begin;

-- add unique(id, provider)
alter table users
    add constraint users_id_provider_uk
        unique (id, provider);

-- session table for session tokens <-> user id
create table if not exists sessions
(
    user_id    text                                      not null,
    provider   sso_provider                              not null,
    hash       text    default md5((random())::text)     not null,
    country    text,
    created_at timestamptz default current_timestamp     not null,
    updated_at timestamptz default current_timestamp     not null,
    constraint sessions_pk
        primary key (hash, user_id),
    constraint sessions_users_id_provider_fk
        foreign key (user_id, provider) references users (id, provider)
            on delete cascade
);

commit;
