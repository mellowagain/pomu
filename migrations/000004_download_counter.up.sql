begin;

alter table videos
    add if not exists downloads
        integer default 0 not null;

commit;
