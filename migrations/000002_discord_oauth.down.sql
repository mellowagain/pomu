begin;

alter table users drop column if exists provider;
drop type sso_provider cascade;

commit;
