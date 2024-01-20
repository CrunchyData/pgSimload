# Most simple example

This example is the simpliest one:

Prerequisites

 - PostgreSQL Server, any version shoud work
 - user has LOGIN capabilities

No creation of tables and others are needed, so there's no need to call for
`-create <create.json>` or such. pgSimload accepts the omission of parameter
`-create`.

This is the most simple example, and here you learn how you can eventually add
a wait in your loops,:

 - `script.one.sql` contains a simple "select 1;"

 - `script.two.sql` contains the same, but with a sleep(1) added

Once pgSimload has been compiled **and** the `config.json` adapted to suit
your needs, this could be used as simple as:

```code
$ pgSimload -config config.json -script script.one.sql
$ pgSimload -config config.json -script script.two.sql
```

If you want to limit the number of loops, you can do that as simply as 

```code
$ pgSimload -config config.json -script script.one.sql -loops 10
```

Alternatively, you can limit the execution time, setting a duration:

```code
$ pgSimload -config config.json -script script.one.sql -time 5s
```

And finally, you can do both at the same time. Whichever happens first will
break the SQL-Loop:

```code
$ pgSimload -config config.json -script script.one.sql -time 1s -loops 20
$ pgSimload -config config.json -script script.one.sql -time 10s -loops 20
```
