-- What connections did and tried, for the operator. Kept past their table,
-- until pruned by age. Ids come from the server, in order.

create table audit (
  id         bigint primary key,
  time       bigint not null, -- unix ms
  table_id   text not null,   -- empty before joining a table
  table_name text not null,
  person     text not null,
  name       text not null,
  ip         text not null,
  conn       text not null,
  kind       text not null,
  error      text not null,   -- empty unless refused
  data       text not null    -- JSON, empty for none
);
create index audit_table on audit (table_id, id);
create index audit_table_name on audit (table_name, id);
create index audit_ip on audit (ip, id);
create index audit_name on audit (name, id);
create index audit_time on audit (time);
