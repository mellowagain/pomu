begin;

drop table if exists sessions;

alter table users drop constraint if exists users_id_provider_uk;

commit;
