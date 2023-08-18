begin;
insert into test.data (a) values ((round(random()*100+1))::integer);
with rnd as (select floor(random()*10000000::integer) as rnd_id) update test.data set insert_date = current_timestamp from rnd where id=rnd_id;
delete from test.data where id = floor(random()*10000000::integer);
commit;
