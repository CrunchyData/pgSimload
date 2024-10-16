# Overview

Welcome to pgSimload !

pgSimload is a tool written in Go, and accepts 3 different modes of execution:

  - **SQL-Loop** mode to execute a script infintely on a given schema of a
    given database with a given user 

  - **Patroni-Watcher** mode to execute a monitoring on a given Patroni
    cluster. This is usefull only if... you run Patroni

  - **Kube-Watcher** mode to have a minimal monitoring of a given PostgreSQL
    cluster in Kubernetes

Given the mode you choose, some parameters are mandatory or not. And the
contexts of executions are different. Please refer to the complete
documentation in
[docs/pgSimload.doc.md](https://github.com/CrunchyData/pgSimload/tree/master/doc).

Alternatively, you can download the [documentation in PDF format](https://github.com/CrunchyData/pgSimload/blob/master/doc/pgSimload.doc.pdf).

# Running, Building and Installing binary

## Running with Go

This is very straightforward if you have Go installed on your system.
You can run the tool with Go from the main directory of the project like:

```code
go run . <parameters...>
go run . -h
```

## Using binaries provided

If you don't have Go installed on your system, you can also just use one of
the binaries provided in [bin/](https://github.com/CrunchyData/pgSimload/tree/master/bin). 

If you want to build your own binary you can build it too, as described in the
next paragraph.

Feedback is welcome in any cases!

## Building binaries

You can use the provided script
[build.sh](https://github.com/CrunchyData/pgSimload/blob/master/build.sh).

```code 
sh build.sh
```

## DEB and RPM packages

We've started tests to build those packages but at the moment, the work hasn't
finish yet. But do you really need this, since pgSimload is a standalone
binary ?

# Usages

This tool can be used in different infrastructures:

  - on the localhost, if a PostgreSQL is running on it
  
  - on any distant stand-alone PostgreSQL or PostgreSQL cluster, in 
    bare-metal of VMs

  - on any PostgreSQL stand-alone PostgreSQL or PostgreSQL cluster
    running in a Kubernetes environment. 

This tool as different usages, and you probably think of some that I haven't
listed here:

  - just initiate a plain `select 1`, a `select count(*) from...`, whatever
    you find usefull. But pgSimload won't get you results back from those
    executions
 
  - insert dummy data (mostly randomly if you know about, mostly,
    `generate_series()` and `random()` PostgreSQL functions) any DB with the
     schema of your choice and the SQL script of your choice
   
    - if your database doesn't have a schema yet, you can create in a
      `create.json` file. Look for examples on how to do that in the
      `examples/SQL-Loop/` directory. It should straightforward. That file is
      **not** mandatory, as pgSimload need at least a `-config <file>` and a
      `-script <file>` to run, in SQL-Loop mode.

    - the SQL script of your choice. For that purpose you create a plain 
      SQL file, where you put everything you want in it. It will be run in an
      implicit transaction, and can contain multiple statements. If you want
      details on how pgSimload runs those statements at once, please read 
      chapter [Multiple Statements in a Simple Query](https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-MULTI-STATEMENT) 
      in the PostgreSQL's documentation.

    - you can set special parameters to the session like `SET
      synchronous_commit TO 'on'` or `SET work_mem TO '12MB'` if you want
      the SQL script's sessions to be tweaked depending your needs. This is
      usefull to compare the performances or behaviour in replication or
      others things. For that you'll have to use the `-session_parameters
      <session_parameters.json>` parameter for pgSimpload. Otherwise, without
      this, every DEFAULT values will of course apply.

    - if you're too lazy to gather those session parameters, you can create
      a template file you can letter modify and adapt to your needs. For that
      pgSimload will create a template file in the name you want, based on a 
      given connection. Look for `-create_gucs_template` in this
      documentation.

    - this "dummy data insertion" is most often used to simulate some 
      write work on a PostgreSQL server (standalone or the primary of a 
      PostgreSQL cluster with a(some) replica(s).

    - the SQL-Loop mode execution can be limitated to:
  
      - a number of loop exections you define thanks to the `-loops <int64>` 
        parameter and/or 
  
      - a given execution time of your choice you can define thanks to the
        `-time duration` parameter, where that duration is expressed with
        or without simple or double-quotes like in "10s", 1m30s or '1h40m'

      - if both parameters are used at the same time, the SQL-Loop will end
        whenever one or the other condition is satisfied

    - the rate of the iterations can be slowed down since version 1.2.0 thanks
      to the `-sleep duration` parameter, where a duration is expressed the
      same way `-time duration` is (see upper). If this parameter is set to 
      anything else that 0, pgSimload will sleep for that amount of time. 
      This is usefull if you want to slow down the SQL-Loop process. It also
      avoid the user to manually add like a `select pg_sleep(1);` at the end
      of the `SQL script` used with `-script`. So it's faster to test
      different values of "sleeping" by recalling the command line and
      changing the value there instead of editing that SQL script...

    - since version 1.4.1, the parameter `-rsleep duration` allows to set 
      a maximum random sleep time of the duration in parameter. This is 
      usefull if you want not all your `-clients <integer>` to be executing
      the `-script` at the exact same time. OR if you prefer the *sleep time* 
      between iterations to be somewhat random and not fixed. 
      If `-sleep` and `-rsleep` are used *both*, then the random sleep time
      will be *added* to the fixed sleep time. As an example a `-sleep 1s`
      with a `-rsleep 1s` will result of a total sleep time *between* 1 and
      2 seconds.
 
  - the SQL-loop mode execution is by default executed with one unique
    PostgreSQL connection to the server. You can execute it with as many
    clients in parallel you want thanks to the `-clients <integer>` parameter
    added in version 1.4.0. So if you want the same SQL script be executed 
    by 3 parallel clients, that is simple as adding `-clients 3` to the
    command line. If you use limitations (`-loops` and/or `-time` and/or
    `-sleep`) and/or special session parameters (`-session_parameters`), those
    will be applied to all clients the same way.

  - test failovers, or what happens when a DB is down: pgSimLoad handles those
    errors. Give it a try: simply shuting down your PostgreSQL server while it
    runs... You'll see it throwing errors, then restarting to load once the
    PostgreSQL server ("primary" if you use replication) is back. 

  - monitor a PostgreSQL cluster that uses Patroni, with the special 
    `--patroni <config.json>` parameter, that has to come with a 
    `--config <config.json>` where the later will use **mandatorily** the
    `postgres` user, because, on that mode, we use a special trick to get the
    primary's name, and this trick can only be done by a superuser in
    PostgreSQL (so it can be something else than `postgres`, if you set
    another superuser).

  - so when testing a PostgreSQL cluster using Patroni, with multiple 
    hosts (a primary and a given number of replicas, synchronous or not),
    usually, pgSimload is run in 2 separate terminals, one to load data,
    and the other, to monitor things in Patroni.

    - note the Patroni-Watcher mode can have added information thanks
      to the `Replication_info` set to `nogucs` or `<list of gucs separated by
      a comma` (e.g "synchronous_standby_names, synchronous_commit, work_mem") in
      the `patroni.json` config file passed as an argument to `-patroni
      <patroni.json>` parameter. If set to `nogucs`, no extra GUCs are shown,
      only the info from `pg_stat_replication` will be

  - monitor a PostgreSQL cluster that runs in Kubernetes, wheter this
    solution uses Patroni or not for HA: this mode only uses some `kubectl` 
    commands to gather only the relevant information to monitor things, like
    who's primary, who's replica, the status of each, etc. This mode has been
    tested against the Postgres Operator (aka PGO), from CrunchyData, and the
    operator from CloudNativePG. You'll find in the `example/Kube-Watcher/` 
    directory proper configuration JSON to use in both cases

  - demo [Crunchy Postgres](https://www.crunchydata.com/products/crunchy-high-availability-postgresql), a fully Open Source based PostgreSQL distribution
    using extensively Ansible 

  - demo [Crunchy Postgres for Kubernetes](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes), a fully Open Source based PostgreSQL 
    distribution to run production workloads in Kubernetes

