# Overview

Welcome to pgSimload !

The actual version of the program is: 

**pgSimload version 1.0.0 - December, 8th 2023**

pgSimload is a tool written in Go, and accepts 2 different modes of execution:

  - **SQL-loop mode** to execute a script infintely on a given schema of a
    given database with a given user 

  - **Patroni-monitoring mode** to execute a monitoring on a given Patroni
    cluster. So you need one of such for this mode to be useful to you

Given the mode you choose, some parameters are mandatory or not. And the
contexts of executions are different. Please refer to the complete
documentation in [docs/pgSimload.doc.md](doc/pgSimload.doc.md)

# Release notes

After 4 months of intensive tests, pgSimload v.1.0.0 is out after the beta
perdiod!

## Version 1.0.0 (December, 8th 2023)

After 3 months of intensive tests, pgSimload v.1.0.0 is out after the beta
period!

What's new?
 - updated Go modules
 - rebuild of binaries
 - tagging version 1.0.0
 - minor fixes in documentation (links)

## Version 1.0.0-beta (July, 24th 2023)

First released version of pgSimload !

# Running, Building and Installing binary

## Running with Go

This is very straightforward if you have Go installed on your system.
You can run the tool with Go from the main directory of the project like:

```code
$ go run main.go <parameters...>
$ go run main.go -h
```

## Using binaries provided

If you don't have Go installed on your system, you can also just use one of
the binaries provided in [bin/](https://github.com/CrunchyData/pgSimload/blob/master/main/bin/)
. If you want to build your own binary you can
build it too, as described in the next paragraph.

Please read carefully [bin/README.md](https://github.com/CrunchyData/pgSimload/blob/master/main/bin/README.md),
where we told you wich binary to use depending on your environment, specially
in Linux.

Note that Mac and Windows versions aren't fully tested at the moment. 

Feedback is welcome in any cases!

## Building binaries

You can use the provided script [build.sh](https://github.com/CrunchyData/pgSimload/blob/master/main/build.sh):

```code 
$ sh build.sh
```

## DEB and RPM packages

We've started tests to build those packages but at the moment, the work hasn't
finish yet. Those packages will be available soon.

# Usages

This tool can be used in different infrastructures:

  - on the localhost, if a PostgreSQL is running on it
  
  - on any distant stand-alone PostgreSQL or PostgreSQL cluster, in 
    bare-metal of VMs

  - on any PostgreSQL stand-alone PostgreSQL or PostgreSQL cluster
    running in a Kubernetes environment. 

This tool can be used in different scenarios:

  - insert dummy data (mostly randomly if you know about, mostly,
    `generate_series()` and `random()` PostgreSQL functions) any DB with the
     schema of your choice and the SQL script of your choice
   
    - if your database doesn't have a schema yet, you can create in a
      `create.json` file. Look for examples on how to do that in the
      `examples/` directory. It should straightforward. That file is **not** 
      mandatory, as pgSimload need at least a `-config <file>` and a `-script
      <file>` to run.

    - the SQL script of your choice. For that purpose you create a plain 
      SQL file, where you put everything you want in it. Beware the parsing
      is really simple, it would probably fail when creating complex things
      like functions in this script.

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
    another superuser)

  - so when testing a PostgreSQL cluster using Patroni, with multiple 
    hosts (a primary and a given number of replicas, synchronous or not),
    usually, pgSimload is run in 2 separate terminals, one to load data,
    and the other, to monitor things in Patroni

    - note the Patroni-monitoring mode can have added information thanks
      to the `Replication_info` set to `nogucs` or `<list of gucs separated by
      a comma` (e.g "synchronous_standby_names, synchronous_commit, work_mem") in
      the `patroni.json` config file passed as an argument to `-patroni <patroni.json>` parameter. 
      If set to `nogucs`, no extra GUCs are shown, only the info from `pg_stat_replication` will be.

  - demo [Crunchy Postgres](https://www.crunchydata.com/products/crunchy-high-availability-postgresql), a fully Open Source based PostgreSQL distribution
    using extensively Ansible

  - demo [Crunchy Postgres for Kubernetes](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes),
    a fully Open Source based PostgreSQL distribution to run production workloads in Kubernetes

For a complete documentation, please have a look at
[docs/pgSimload.doc.md](doc/pgSimload.doc.md)
online or alternativately download the [PDF
Version](https://github.com/CrunchyData/pgSimload/blob/master/doc/pgSimload.doc.pdf)
of the documentation for offline read of 20 pages.

This documentation contains **examples** explanation, and full, comprehensive
reference on flags and parameters the program accepts.

-- jpa
