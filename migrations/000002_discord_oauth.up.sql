begin;

-- https://stackoverflow.com/a/48382296/11494565
do
$$
    begin
        create type sso_provider as enum ('google', 'discord');
    exception
        when duplicate_object then null;
    end
$$;

alter table users add if not exists provider sso_provider default 'google' not null;

commit;
