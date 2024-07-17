# Examples given

You can find several examples of usage in the main directory of the project
under `examples/`.

## `examples/SQL-Loop/simple/`

This example is the simpliest example of the SQL-Loop mode!

Prerequisites

 - PostgreSQL Server, any version shoud work
 - user has LOGIN capabilities

No creation of tables and others are needed, so there's no need to call for
`-create <create.json>` or such. pgSimload accepts the omission of parameter 
`-create`.

`script.sql` contains a simple "select 1;". There's also a comment starting
with `--`, like in ... plain SQL. So that means this file is just the SQL
script you want it to be. Simple as that.

Once pgSimload has been compiled **and** the `config.json` adapted to suit
your needs, this could be used as simple as:

```code
pgSimload -config config.json -script script.sql
```

You can throtle it down asking pgSimload to wait for 1 second and a half like
this:

```code
pgSimload -config config.json -script script.sql -sleep 1s500ms
```

If you want to limit the number of loops, you can do that as simply as

```code
pgSimload -config config.json -script script.sql -loops 10
```

Alternatively, you can limit the execution time, setting a duration:

```code
pgSimload -config config.json -script script.sql -time 5s
```
You can do both at the same time. Whichever happens first will
break the SQL-Loop:

```code
pgSimload -config config.json -script script.sql -time 1s -loops 20
pgSimload -config config.json -script script.sql -time 10s -loops 20
```

## `examples/SQL-Loop/PG_15_Merge_command/`

This example is to test the MERGE command first introduced in PostgreSQL
version 15.

Prerequisites

 - PostgreSQL Server version 15+ stand-alone or not
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
pgSimload -config config.json -create create.json -script script.sql
```

The `watcher.sh` is a plain psql into watch to get some live stats on the
database. You may have to adapt it to match your usage. I've added 2 flavours.

The first show some data, nice to have in a separate terminal (use
[tilix](https://gnunn1.github.io/tilix-web/) while you demo!):

```code
sh watcher.sh query
```

The second shows a nice histogram of the data, the query is slightly more
complex and heaven tho:

```code
sh watcher.sh histogram
```

## `examples/SQL-Loop/testdb/`

This is another example that shows one can:

 - create multiple different `create.json` files to match different scenarios,
   adding different things like in `create.json`, `create.delete.json`,
`create.delete.vacuum.json`, etc. to pass to the paramter `-create`
 - create multiple different `script.sql`, `insert.sql`, etc.. to pass to the
   parameter `-create`

Obviously, that `delete from test.data;` is just for the example, if you
really want to delete all data from a table, in the real world, you need to
use [truncate
data](https://www.postgresql.org/docs/current/sql-truncate.html)!

If you have a PostgreSQL *cluster* where you want to test as an example:

 - write activity to the primary and
 - read activity to the secondary

Then you'll need 2 different files for credentials one to your primary, on
let's say port 5432, another one to your secondary (or pool of secondaries, if
you're using `pgBouncer` on a different port, or just `HAProxy` or anything
else to balance to different PostgreSQL replicas, on let's say, port 5433).

You'll need also 2 different SQL script files to run read/write operations on
the primary, and obviously, read/only operations to the secondary (or group of
secondaries).

Finaly, you will have to run twice pgSimload, in 2 different terminals, to
handle boths scenarios at the same time.

We give here a special example of the file `session_parameters.json` (you can
name that like you want), as for you to use the special `-session_parameters
<session_parameter.json>` if you want to modify the parameters of the session
in which the script.sql queries will exectute. You can use this to set special
values to a lot of configuration parameters that PostgreSQL allows to change
within a session. As an example: `work_mem`, `synchronous_commit`, etc.

