begin;

alter table videos ADD COLUMN downloads int DEFAULT 0 not null;

commit;