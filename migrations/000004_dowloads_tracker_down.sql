begin;

alter table videos drop column if EXISTS downloads;

commit;