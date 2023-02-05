begin;

alter table videos ADD COLUMN if not EXISTS downloads int DEFAULT 0 not null;

commit;