# Example to test pgSimLoad and MERGE command in PG 15.x

This example is to test new MERGE command in PG 15.x

Prerequisites

 - PostgreSQL Server version 15+
 - a user called owning a schema name "test" (please adapt config.json file to
   match your needs here)
 - user has LOGIN capabilities

Creates a schema with 3 tables in `create.json`.

`script.sql` will:

 - create sample data in `test.station_data_new`
 - merge that data in `test.station_data_actual`
 - merge `test.station_data_actual` into `test.station_data_history`

As per [jpa's blog article on MERGE](https://www.crunchydata.com/blog/a-look-at-postgres-15-merge-command-with-examples)

Once pgSimload has been compiled and the binary placed in some dir your
`$PATH` points to, this could be used as simple as:

```code
$ pgSimload -config config.json -create create.json -script script.sql
```

The `watcher.sh` is a plain psql into watch to get some live stats on the
database. You may have to adapt it to match your usage. I added 2 flavours.

The first show some data, nice to have in a separate terminal (use
[tilix](pgSimload -config config.json -script script.two_liner.sql)!)
while you demo:

```code
$ sh watcher.sh query
```

The second shows a nice histogram of the data, the query is slightly more
complex and heaven tho:

```code
$ sh watcher.sh histogram
```
