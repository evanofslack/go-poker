-- Runs on Postgres and SQLite: JSON is text and times are unix ms.

create table tables (
  id           text primary key,
  name         text not null,
  version      integer not null, -- table.Version of state
  state        text not null,    -- table.State as JSON, without its hands and logs
  created_at   bigint not null,
  updated_at   bigint not null,
  closed_at    bigint,           -- null while open
  close_reason text
);
create unique index tables_open_name on tables (name) where closed_at is null;

create table hands (
  table_id text not null,
  hand     integer not null,
  record   text not null, -- table.HandRecord
  stats    text not null, -- table.HandStats
  primary key (table_id, hand)
);

create table logs (
  table_id text not null,
  id       integer not null,
  message  text not null,
  time     bigint not null,
  primary key (table_id, id)
);
