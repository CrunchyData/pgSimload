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
