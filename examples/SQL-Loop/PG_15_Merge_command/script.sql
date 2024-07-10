begin;

truncate table test.station_data_new;

with measures as (
  select *
  from generate_series(round(random()*1000+1)::integer,round(random()*1000)::integer+100)
)
insert into test.station_data_new (
    station_id
  , a
  , b
)
select
    generate_series
  , round(random()*1000+1)
  , round(random()*1000+1)
from
  measures;

merge into test.station_data_actual sda
using test.station_data_new sdn
on sda.station_id = sdn.station_id
when matched then
  update set a = sdn.a, b = sdn.b, updated = default
when not matched then
  insert (station_id, a, b)
  values (sdn.station_id, sdn.a, sdn.b);
merge into test.station_data_history sdh
using test.station_data_actual sda
on ( sda.station_id = sdh.station_id
 and sda.updated    = sdh.updated)
when matched then
  do nothing
when not matched then
  insert (station_id, a, b, updated)
  values (sda.station_id, sda.a, sda.b, sda.updated);

commit;
