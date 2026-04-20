create table if not exists public.user_profiles (
  id uuid primary key references auth.users (id) on delete cascade,
  push_token text,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  constraint user_profiles_push_token_not_blank check (
    push_token is null
    or char_length(trim(push_token)) > 0
  )
);

create index if not exists user_profiles_push_token_idx
on public.user_profiles (push_token)
where push_token is not null;

drop trigger if exists user_profiles_set_updated_at on public.user_profiles;
create trigger user_profiles_set_updated_at
before update on public.user_profiles
for each row
execute function public.set_updated_at();

insert into public.user_profiles (id)
select users.id
from auth.users as users
on conflict (id) do nothing;

create or replace function public.ensure_user_profile()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
  insert into public.user_profiles (id)
  values (new.id)
  on conflict (id) do nothing;

  return new;
end;
$$;

drop trigger if exists ensure_user_profile_after_auth_insert on auth.users;
create trigger ensure_user_profile_after_auth_insert
after insert on auth.users
for each row
execute function public.ensure_user_profile();

alter table public.user_profiles enable row level security;

create policy "user_profiles_select_own"
on public.user_profiles
for select
using (auth.uid() = id);

create policy "user_profiles_insert_own"
on public.user_profiles
for insert
with check (auth.uid() = id);

create policy "user_profiles_update_own"
on public.user_profiles
for update
using (auth.uid() = id)
with check (auth.uid() = id);
