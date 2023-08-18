#!/bin/bash

query="
select * 
from test.station_data_actual 
where station_id in
(10,30,40,50,60) 
order by station_id; 
select station_id,count(*) 
from test.station_data_history 
where station_id in (10,30,40,50,60) 
group by 1 
order by 1; 
select count(*) as station_history_count 
from test.station_data_history"

# per https://tapoueh.org/blog/2014/02/postgresql-aggregates-and-histograms/
# Thanks Dimitri :-)
cat > histogram.sql <<EOF
with stations_stats as (
    select min(station_id) as min,
           max(station_id) as max
      from test.station_data_history
),
     histogram as (
   select width_bucket(station_id, min, max, 20) as bucket,
          int4range(min(station_id), max(station_id), '[]') as range,
          count(*) as freq
     from test.station_data_history, stations_stats
 group by bucket
 order by bucket
)
 select bucket, range, freq,
        repeat('■',
               (   freq::float
                 / max(freq) over()
                 * 30
               )::int
        ) as bar
   from histogram;
EOF

case "$1" in
  "query")
    watch -n 2 "psql --pset footer=off -q -c '${query}' stations"
    ;;
  "histogram")
    watch -n 5 "psql --pset footer=off -q -f histogram.sql stations"
    ;;
  *)
    echo "Usage : $0 (query|histogram]"
    echo ""
    exit 0
    ;;
esac

echo $query

