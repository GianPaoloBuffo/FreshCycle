create extension if not exists "pgcrypto";

create or replace function public.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = timezone('utc', now());
  return new;
end;
$$;

create table if not exists public.garments (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users (id) on delete cascade,
  name text not null,
  category text,
  primary_color text,
  wash_temperature_c integer,
  care_instructions text[] not null default '{}'::text[],
  label_image_path text,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  constraint garments_name_not_blank check (char_length(trim(name)) > 0),
  constraint garments_wash_temperature_range check (
    wash_temperature_c is null
    or wash_temperature_c between 0 and 95
  )
);

create table if not exists public.laundry_schedules (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users (id) on delete cascade,
  name text not null,
  recurrence text not null,
  reminder_enabled boolean not null default true,
  starts_on date,
  last_completed_at timestamptz,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  constraint laundry_schedules_name_not_blank check (char_length(trim(name)) > 0),
  constraint laundry_schedules_recurrence_not_blank check (char_length(trim(recurrence)) > 0)
);

create table if not exists public.schedule_garments (
  schedule_id uuid not null references public.laundry_schedules (id) on delete cascade,
  garment_id uuid not null references public.garments (id) on delete cascade,
  created_at timestamptz not null default timezone('utc', now()),
  primary key (schedule_id, garment_id)
);

create index if not exists garments_user_id_idx on public.garments (user_id);
create index if not exists garments_category_idx on public.garments (user_id, category);
create index if not exists garments_primary_color_idx on public.garments (user_id, primary_color);
create index if not exists garments_wash_temperature_idx on public.garments (user_id, wash_temperature_c);
create index if not exists laundry_schedules_user_id_idx on public.laundry_schedules (user_id);
create index if not exists schedule_garments_garment_id_idx on public.schedule_garments (garment_id);

drop trigger if exists garments_set_updated_at on public.garments;
create trigger garments_set_updated_at
before update on public.garments
for each row
execute function public.set_updated_at();

drop trigger if exists laundry_schedules_set_updated_at on public.laundry_schedules;
create trigger laundry_schedules_set_updated_at
before update on public.laundry_schedules
for each row
execute function public.set_updated_at();

alter table public.garments enable row level security;
alter table public.laundry_schedules enable row level security;
alter table public.schedule_garments enable row level security;

create policy "garments_select_own"
on public.garments
for select
using (auth.uid() = user_id);

create policy "garments_insert_own"
on public.garments
for insert
with check (auth.uid() = user_id);

create policy "garments_update_own"
on public.garments
for update
using (auth.uid() = user_id)
with check (auth.uid() = user_id);

create policy "garments_delete_own"
on public.garments
for delete
using (auth.uid() = user_id);

create policy "laundry_schedules_select_own"
on public.laundry_schedules
for select
using (auth.uid() = user_id);

create policy "laundry_schedules_insert_own"
on public.laundry_schedules
for insert
with check (auth.uid() = user_id);

create policy "laundry_schedules_update_own"
on public.laundry_schedules
for update
using (auth.uid() = user_id)
with check (auth.uid() = user_id);

create policy "laundry_schedules_delete_own"
on public.laundry_schedules
for delete
using (auth.uid() = user_id);

create policy "schedule_garments_select_own"
on public.schedule_garments
for select
using (
  exists (
    select 1
    from public.laundry_schedules schedules
    where schedules.id = schedule_id
      and schedules.user_id = auth.uid()
  )
);

create policy "schedule_garments_insert_own"
on public.schedule_garments
for insert
with check (
  exists (
    select 1
    from public.laundry_schedules schedules
    where schedules.id = schedule_id
      and schedules.user_id = auth.uid()
  )
  and exists (
    select 1
    from public.garments garments
    where garments.id = garment_id
      and garments.user_id = auth.uid()
  )
);

create policy "schedule_garments_delete_own"
on public.schedule_garments
for delete
using (
  exists (
    select 1
    from public.laundry_schedules schedules
    where schedules.id = schedule_id
      and schedules.user_id = auth.uid()
  )
);
