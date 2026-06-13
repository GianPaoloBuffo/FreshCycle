create table if not exists public.scan_retention_policies (
  environment text primary key,
  preferred_image_scope text not null default 'label_crop',
  store_full_images boolean not null default false,
  image_retention interval not null,
  ocr_text_retention interval not null,
  image_hash_retention interval not null,
  annotation_retention interval not null,
  review_record_retention interval not null,
  debug_storage_allowed boolean not null default false,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  constraint scan_retention_environment_check check (environment in ('production', 'debug', 'test')),
  constraint scan_retention_image_scope_check check (preferred_image_scope in ('label_crop', 'unknown')),
  constraint scan_retention_non_negative_check check (
    image_retention >= interval '0 seconds'
    and ocr_text_retention >= interval '0 seconds'
    and image_hash_retention >= interval '0 seconds'
    and annotation_retention >= interval '0 seconds'
    and review_record_retention >= interval '0 seconds'
  ),
  constraint scan_retention_no_production_full_images check (
    environment <> 'production' or store_full_images = false
  )
);

insert into public.scan_retention_policies (
  environment,
  preferred_image_scope,
  store_full_images,
  image_retention,
  ocr_text_retention,
  image_hash_retention,
  annotation_retention,
  review_record_retention,
  debug_storage_allowed
)
values
  ('production', 'label_crop', false, interval '30 days', interval '30 days', interval '180 days', interval '400 days', interval '180 days', false),
  ('debug', 'label_crop', false, interval '14 days', interval '14 days', interval '60 days', interval '180 days', interval '90 days', true),
  ('test', 'label_crop', false, interval '1 day', interval '1 day', interval '7 days', interval '30 days', interval '30 days', true)
on conflict (environment) do update
set
  preferred_image_scope = excluded.preferred_image_scope,
  store_full_images = excluded.store_full_images,
  image_retention = excluded.image_retention,
  ocr_text_retention = excluded.ocr_text_retention,
  image_hash_retention = excluded.image_hash_retention,
  annotation_retention = excluded.annotation_retention,
  review_record_retention = excluded.review_record_retention,
  debug_storage_allowed = excluded.debug_storage_allowed,
  updated_at = timezone('utc', now());

create table if not exists public.scan_events (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users (id) on delete cascade,
  environment text not null default 'production',
  outcome text not null,
  error_code text,
  capture_source text,
  image_scope text not null default 'label_crop',
  image_hash text,
  image_mime_type text,
  image_size_bytes integer,
  full_photo_stored boolean not null default false,
  has_client_ocr boolean not null default false,
  ocr_text_hash text,
  client_symbol_count integer not null default 0,
  capture_quality_score numeric,
  capture_quality_issues text[] not null default '{}'::text[],
  provider text,
  route text,
  cache_hit boolean not null default false,
  paid_fallback_used boolean not null default false,
  fallback_calls_avoided integer not null default 0,
  confidence numeric,
  needs_user_confirmation boolean not null default false,
  uncertain_fields text[] not null default '{}'::text[],
  routing_reasons text[] not null default '{}'::text[],
  symbol_classes text[] not null default '{}'::text[],
  field_confidences jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  retention_expires_at timestamptz not null default (timezone('utc', now()) + interval '180 days'),
  constraint scan_events_environment_check check (environment in ('production', 'debug', 'test')),
  constraint scan_events_outcome_check check (outcome in ('success', 'failure')),
  constraint scan_events_image_scope_check check (image_scope in ('label_crop', 'full_photo', 'unknown')),
  constraint scan_events_client_symbol_count_check check (client_symbol_count >= 0),
  constraint scan_events_image_size_check check (image_size_bytes is null or image_size_bytes >= 0),
  constraint scan_events_confidence_check check (confidence is null or confidence between 0 and 1),
  constraint scan_events_capture_quality_score_check check (capture_quality_score is null or capture_quality_score between 0 and 1),
  constraint scan_events_no_production_full_photo_storage check (
    environment <> 'production' or full_photo_stored = false
  )
);

create table if not exists public.scan_review_queue (
  id uuid primary key default gen_random_uuid(),
  scan_event_id uuid references public.scan_events (id) on delete set null,
  user_id uuid not null references auth.users (id) on delete cascade,
  status text not null default 'queued',
  priority numeric not null default 0,
  review_reasons text[] not null default '{}'::text[],
  cropped_label_image_path text,
  image_hash text,
  ocr_output text,
  detector_output jsonb not null default '{}'::jsonb,
  model_result jsonb not null default '{}'::jsonb,
  final_user_correction jsonb,
  decision text,
  decided_at timestamptz,
  privacy_delete_requested_at timestamptz,
  redacted_at timestamptz,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  retention_expires_at timestamptz not null default (timezone('utc', now()) + interval '180 days'),
  constraint scan_review_status_check check (
    status in ('queued', 'in_review', 'accepted', 'corrected', 'needs_label', 'discarded', 'privacy_deleted', 'redacted')
  ),
  constraint scan_review_decision_check check (
    decision is null or decision in ('accept', 'correct', 'needs_label', 'discard', 'privacy_delete')
  ),
  constraint scan_review_priority_check check (priority between 0 and 1)
);

create table if not exists public.scan_review_decisions (
  id uuid primary key default gen_random_uuid(),
  review_queue_id uuid references public.scan_review_queue (id) on delete set null,
  scan_event_id uuid references public.scan_events (id) on delete set null,
  user_id uuid not null references auth.users (id) on delete cascade,
  decision text not null,
  corrected_fields text[] not null default '{}'::text[],
  field_corrections jsonb not null default '{}'::jsonb,
  final_user_correction jsonb not null default '{}'::jsonb,
  notes text,
  created_at timestamptz not null default timezone('utc', now()),
  constraint scan_review_decisions_decision_check check (
    decision in ('accept', 'correct', 'needs_label', 'discard', 'privacy_delete')
  )
);

create table if not exists public.scan_annotation_examples (
  id uuid primary key default gen_random_uuid(),
  review_queue_id uuid references public.scan_review_queue (id) on delete set null,
  user_id uuid not null references auth.users (id) on delete cascade,
  dataset_version text not null default 'unassigned',
  image_hash text,
  cropped_label_image_path text,
  source text not null default 'review_queue',
  status text not null default 'candidate',
  priority numeric not null default 0,
  uncommon_classes text[] not null default '{}'::text[],
  class_counts jsonb not null default '{}'::jsonb,
  annotation_quality_score numeric,
  inter_annotator_agreement numeric,
  imbalance_bucket text,
  privacy_delete_requested_at timestamptz,
  redacted_at timestamptz,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  retention_expires_at timestamptz not null default (timezone('utc', now()) + interval '400 days'),
  constraint scan_annotation_source_check check (source in ('review_queue', 'manual_import', 'active_learning')),
  constraint scan_annotation_status_check check (
    status in ('candidate', 'labeling', 'labeled', 'qa_passed', 'qa_failed', 'removed', 'privacy_deleted')
  ),
  constraint scan_annotation_priority_check check (priority between 0 and 1),
  constraint scan_annotation_quality_check check (annotation_quality_score is null or annotation_quality_score between 0 and 1),
  constraint scan_annotation_agreement_check check (inter_annotator_agreement is null or inter_annotator_agreement between 0 and 1)
);

alter table public.scan_review_queue
add column if not exists annotation_example_id uuid references public.scan_annotation_examples (id) on delete set null;

create table if not exists public.scan_symbol_annotations (
  id uuid primary key default gen_random_uuid(),
  annotation_example_id uuid not null references public.scan_annotation_examples (id) on delete cascade,
  class text not null,
  label text,
  bounding_box jsonb not null,
  modifiers text[] not null default '{}'::text[],
  annotator_user_id uuid references auth.users (id) on delete set null,
  quality_status text not null default 'pending',
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  constraint scan_symbol_annotations_class_not_blank check (char_length(trim(class)) > 0),
  constraint scan_symbol_annotations_quality_check check (quality_status in ('pending', 'accepted', 'rejected', 'needs_review'))
);

create table if not exists public.detector_model_iterations (
  id uuid primary key default gen_random_uuid(),
  version text not null unique,
  dataset_version text not null,
  artifact_type text not null default 'detector',
  status text not null default 'planned',
  train_config jsonb not null default '{}'::jsonb,
  artifact_paths jsonb not null default '{}'::jsonb,
  metrics jsonb not null default '{}'::jsonb,
  model_size_bytes bigint,
  median_latency_ms numeric,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  promoted_at timestamptz,
  constraint detector_model_iterations_version_not_blank check (char_length(trim(version)) > 0),
  constraint detector_model_iterations_dataset_not_blank check (char_length(trim(dataset_version)) > 0),
  constraint detector_model_iterations_status_check check (
    status in ('planned', 'training', 'evaluating', 'promoted', 'rejected', 'rolled_back')
  ),
  constraint detector_model_iterations_size_check check (model_size_bytes is null or model_size_bytes >= 0),
  constraint detector_model_iterations_latency_check check (median_latency_ms is null or median_latency_ms >= 0)
);

create table if not exists public.detector_evaluation_runs (
  id uuid primary key default gen_random_uuid(),
  model_iteration_id uuid references public.detector_model_iterations (id) on delete set null,
  dataset_version text not null,
  holdout_name text not null,
  scanner_version text,
  accuracy numeric,
  false_positive_count integer not null default 0,
  false_negative_count integer not null default 0,
  median_latency_ms numeric,
  model_size_bytes bigint,
  passed boolean not null default false,
  metrics jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default timezone('utc', now()),
  constraint detector_evaluation_dataset_not_blank check (char_length(trim(dataset_version)) > 0),
  constraint detector_evaluation_holdout_not_blank check (char_length(trim(holdout_name)) > 0),
  constraint detector_evaluation_accuracy_check check (accuracy is null or accuracy between 0 and 1),
  constraint detector_evaluation_fp_check check (false_positive_count >= 0),
  constraint detector_evaluation_fn_check check (false_negative_count >= 0),
  constraint detector_evaluation_latency_check check (median_latency_ms is null or median_latency_ms >= 0),
  constraint detector_evaluation_size_check check (model_size_bytes is null or model_size_bytes >= 0)
);

create table if not exists public.scan_field_accuracy_events (
  id uuid primary key default gen_random_uuid(),
  scan_event_id uuid references public.scan_events (id) on delete set null,
  review_decision_id uuid references public.scan_review_decisions (id) on delete set null,
  model_iteration_id uuid references public.detector_model_iterations (id) on delete set null,
  user_id uuid not null references auth.users (id) on delete cascade,
  field_name text not null,
  predicted_value text,
  corrected_value text,
  error_source text not null default 'unknown',
  confidence numeric,
  calibrated_bucket text,
  was_correct boolean not null default false,
  created_at timestamptz not null default timezone('utc', now()),
  constraint scan_field_accuracy_field_name_check check (
    field_name in (
      'wash_temperature',
      'wash_cycle',
      'hand_wash',
      'bleach',
      'tumble_dry',
      'natural_drying',
      'iron',
      'dry_clean',
      'wet_clean'
    )
  ),
  constraint scan_field_accuracy_error_source_check check (
    error_source in ('ocr', 'symbol_detection', 'rule_interpretation', 'user_override', 'unknown')
  ),
  constraint scan_field_accuracy_confidence_check check (confidence is null or confidence between 0 and 1)
);

create table if not exists public.active_learning_candidates (
  id uuid primary key default gen_random_uuid(),
  scan_event_id uuid references public.scan_events (id) on delete set null,
  review_queue_id uuid references public.scan_review_queue (id) on delete set null,
  annotation_example_id uuid references public.scan_annotation_examples (id) on delete set null,
  user_id uuid not null references auth.users (id) on delete cascade,
  status text not null default 'candidate',
  priority_score numeric not null default 0,
  priority_reasons text[] not null default '{}'::text[],
  source_signals text[] not null default '{}'::text[],
  uncommon_classes text[] not null default '{}'::text[],
  detector_version text,
  model_version_after_training text,
  accuracy_delta jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default timezone('utc', now()),
  updated_at timestamptz not null default timezone('utc', now()),
  constraint active_learning_status_check check (
    status in ('candidate', 'selected', 'annotated', 'trained', 'closed', 'privacy_deleted')
  ),
  constraint active_learning_priority_check check (priority_score between 0 and 1)
);

create index if not exists scan_events_user_created_idx on public.scan_events (user_id, created_at desc);
create index if not exists scan_events_review_flags_idx on public.scan_events (user_id, needs_user_confirmation, confidence);
create index if not exists scan_events_image_hash_idx on public.scan_events (image_hash) where image_hash is not null;
create index if not exists scan_review_queue_user_status_idx on public.scan_review_queue (user_id, status, priority desc, created_at desc);
create index if not exists scan_review_queue_scan_event_idx on public.scan_review_queue (scan_event_id) where scan_event_id is not null;
create index if not exists scan_review_decisions_user_created_idx on public.scan_review_decisions (user_id, created_at desc);
create index if not exists scan_annotation_examples_status_idx on public.scan_annotation_examples (status, priority desc, created_at desc);
create index if not exists scan_annotation_examples_user_idx on public.scan_annotation_examples (user_id, created_at desc);
create index if not exists scan_symbol_annotations_example_idx on public.scan_symbol_annotations (annotation_example_id);
create index if not exists detector_model_iterations_status_idx on public.detector_model_iterations (status, created_at desc);
create index if not exists detector_evaluation_runs_model_idx on public.detector_evaluation_runs (model_iteration_id, created_at desc);
create index if not exists scan_field_accuracy_user_field_idx on public.scan_field_accuracy_events (user_id, field_name, created_at desc);
create index if not exists active_learning_candidates_status_idx on public.active_learning_candidates (status, priority_score desc, created_at desc);
create index if not exists active_learning_candidates_user_idx on public.active_learning_candidates (user_id, created_at desc);

drop trigger if exists scan_retention_policies_set_updated_at on public.scan_retention_policies;
create trigger scan_retention_policies_set_updated_at
before update on public.scan_retention_policies
for each row
execute function public.set_updated_at();

drop trigger if exists scan_events_set_updated_at on public.scan_events;
create trigger scan_events_set_updated_at
before update on public.scan_events
for each row
execute function public.set_updated_at();

drop trigger if exists scan_review_queue_set_updated_at on public.scan_review_queue;
create trigger scan_review_queue_set_updated_at
before update on public.scan_review_queue
for each row
execute function public.set_updated_at();

drop trigger if exists scan_annotation_examples_set_updated_at on public.scan_annotation_examples;
create trigger scan_annotation_examples_set_updated_at
before update on public.scan_annotation_examples
for each row
execute function public.set_updated_at();

drop trigger if exists scan_symbol_annotations_set_updated_at on public.scan_symbol_annotations;
create trigger scan_symbol_annotations_set_updated_at
before update on public.scan_symbol_annotations
for each row
execute function public.set_updated_at();

drop trigger if exists detector_model_iterations_set_updated_at on public.detector_model_iterations;
create trigger detector_model_iterations_set_updated_at
before update on public.detector_model_iterations
for each row
execute function public.set_updated_at();

drop trigger if exists active_learning_candidates_set_updated_at on public.active_learning_candidates;
create trigger active_learning_candidates_set_updated_at
before update on public.active_learning_candidates
for each row
execute function public.set_updated_at();

create or replace view public.scan_field_accuracy_summary
with (security_invoker = true)
as
select
  user_id,
  field_name,
  date_trunc('day', created_at)::date as accuracy_day,
  count(*)::integer as sample_count,
  avg(case when was_correct then 1 else 0 end)::numeric as accuracy,
  avg(confidence)::numeric as average_confidence,
  count(*) filter (where error_source = 'ocr')::integer as ocr_error_count,
  count(*) filter (where error_source = 'symbol_detection')::integer as symbol_detection_error_count,
  count(*) filter (where error_source = 'rule_interpretation')::integer as rule_interpretation_error_count,
  count(*) filter (where error_source = 'user_override')::integer as user_correction_count
from public.scan_field_accuracy_events
group by user_id, field_name, date_trunc('day', created_at)::date;

alter table public.scan_retention_policies enable row level security;
alter table public.scan_events enable row level security;
alter table public.scan_review_queue enable row level security;
alter table public.scan_review_decisions enable row level security;
alter table public.scan_annotation_examples enable row level security;
alter table public.scan_symbol_annotations enable row level security;
alter table public.detector_model_iterations enable row level security;
alter table public.detector_evaluation_runs enable row level security;
alter table public.scan_field_accuracy_events enable row level security;
alter table public.active_learning_candidates enable row level security;

grant select on public.scan_retention_policies to authenticated;
grant select, insert, update, delete on public.scan_events to authenticated;
grant select, insert, update, delete on public.scan_review_queue to authenticated;
grant select, insert, update, delete on public.scan_review_decisions to authenticated;
grant select, insert, update, delete on public.scan_annotation_examples to authenticated;
grant select, insert, update, delete on public.scan_symbol_annotations to authenticated;
grant select on public.detector_model_iterations to authenticated;
grant select on public.detector_evaluation_runs to authenticated;
grant select, insert on public.scan_field_accuracy_events to authenticated;
grant select, insert, update on public.active_learning_candidates to authenticated;
grant select on public.scan_field_accuracy_summary to authenticated;

create policy "scan_retention_policies_select_authenticated"
on public.scan_retention_policies
for select
to authenticated
using (true);

create policy "scan_events_select_own"
on public.scan_events
for select
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_events_insert_own"
on public.scan_events
for insert
to authenticated
with check ((select auth.uid()) = user_id);

create policy "scan_events_update_own"
on public.scan_events
for update
to authenticated
using ((select auth.uid()) = user_id)
with check ((select auth.uid()) = user_id);

create policy "scan_review_queue_select_own"
on public.scan_review_queue
for select
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_review_queue_insert_own"
on public.scan_review_queue
for insert
to authenticated
with check ((select auth.uid()) = user_id);

create policy "scan_review_queue_update_own"
on public.scan_review_queue
for update
to authenticated
using ((select auth.uid()) = user_id)
with check ((select auth.uid()) = user_id);

create policy "scan_review_queue_delete_own"
on public.scan_review_queue
for delete
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_review_decisions_select_own"
on public.scan_review_decisions
for select
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_review_decisions_insert_own"
on public.scan_review_decisions
for insert
to authenticated
with check ((select auth.uid()) = user_id);

create policy "scan_annotation_examples_select_own"
on public.scan_annotation_examples
for select
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_annotation_examples_insert_own"
on public.scan_annotation_examples
for insert
to authenticated
with check ((select auth.uid()) = user_id);

create policy "scan_annotation_examples_update_own"
on public.scan_annotation_examples
for update
to authenticated
using ((select auth.uid()) = user_id)
with check ((select auth.uid()) = user_id);

create policy "scan_annotation_examples_delete_own"
on public.scan_annotation_examples
for delete
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_symbol_annotations_select_own"
on public.scan_symbol_annotations
for select
to authenticated
using (
  exists (
    select 1
    from public.scan_annotation_examples examples
    where examples.id = annotation_example_id
      and examples.user_id = (select auth.uid())
  )
);

create policy "scan_symbol_annotations_insert_own"
on public.scan_symbol_annotations
for insert
to authenticated
with check (
  exists (
    select 1
    from public.scan_annotation_examples examples
    where examples.id = annotation_example_id
      and examples.user_id = (select auth.uid())
  )
);

create policy "scan_symbol_annotations_update_own"
on public.scan_symbol_annotations
for update
to authenticated
using (
  exists (
    select 1
    from public.scan_annotation_examples examples
    where examples.id = annotation_example_id
      and examples.user_id = (select auth.uid())
  )
)
with check (
  exists (
    select 1
    from public.scan_annotation_examples examples
    where examples.id = annotation_example_id
      and examples.user_id = (select auth.uid())
  )
);

create policy "scan_symbol_annotations_delete_own"
on public.scan_symbol_annotations
for delete
to authenticated
using (
  exists (
    select 1
    from public.scan_annotation_examples examples
    where examples.id = annotation_example_id
      and examples.user_id = (select auth.uid())
  )
);

create policy "detector_model_iterations_select_authenticated"
on public.detector_model_iterations
for select
to authenticated
using (true);

create policy "detector_evaluation_runs_select_authenticated"
on public.detector_evaluation_runs
for select
to authenticated
using (true);

create policy "scan_field_accuracy_events_select_own"
on public.scan_field_accuracy_events
for select
to authenticated
using ((select auth.uid()) = user_id);

create policy "scan_field_accuracy_events_insert_own"
on public.scan_field_accuracy_events
for insert
to authenticated
with check ((select auth.uid()) = user_id);

create policy "active_learning_candidates_select_own"
on public.active_learning_candidates
for select
to authenticated
using ((select auth.uid()) = user_id);

create policy "active_learning_candidates_insert_own"
on public.active_learning_candidates
for insert
to authenticated
with check ((select auth.uid()) = user_id);

create policy "active_learning_candidates_update_own"
on public.active_learning_candidates
for update
to authenticated
using ((select auth.uid()) = user_id)
with check ((select auth.uid()) = user_id);
