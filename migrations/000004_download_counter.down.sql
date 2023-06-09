begin;

alter table videos
    drop column if exists downloads;

commit;
