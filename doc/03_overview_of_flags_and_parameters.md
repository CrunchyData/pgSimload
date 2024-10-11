# Overview of flags and parameters

Those tabulars show basic information about flags and parameters. For full
documentation, please read next chapter "Reference : parameters and flags".

## All modes : flags

All flags are optional and intended to run alone, except `silent`.

| Name      |  Description                             |
| :---      |    :-----                                |
| `contact` | Shows author name and email              |
| `help`    | Shows some help                          |
| `license` | Shows license                            |
| `silent`  | Don't show the start banner of pgSimload |
| `version` | Shows current version of pgSimload       |

## SQL-Loop mode : parameters

| Name           |  Mandatory         | Optional           | Value expected  | Description                                   |
| :---           |    :-----:         |  :----:            |  :---:          |    :----                                      |
| `clients`      |                    | **X**              | integer         | Sets the number of parallel runs of the SQL-Loop |
| `config`       | **X**              |                    | JSON file       | Sets the PG connexion string (any user)       |
| `create`       |                    | **X**              | JSON file       | Sets the SQL DDL to run once prior main loop  |
| `script`       | **X**              |                    | SQL text file   | Sets the script to run inside the loop        |
| `session_parameters` |                    | **X**              | JSON file       | Sets special session configuration parameters |
| `loops`        |                    | **X**              | integer         | Sets the number of loops to execute before exiting  |
| `time`         |                    | **X**              | duration        | Sets the total execution time before exiting |
| `sleep`        |                    | **X**              | duration        | Sets a sleep duration between 2 iterations of the SQL-Loop |

## Kube-Watcher mode : parameters

| Name       |  Mandatory         | Optional           | Value expected  | Description                                  |
| :---       |    :-----:         |  :----:            |  :---:          |    :----                                     |
| `kube`     | **X**              |                    | JSON file       | Sets parameters for this special mode        |

## Patroni-Watcher mode : parameters

| Name       |  Mandatory         | Optional           | Value expected  | Description                                  |
| :---       |    :-----:         |  :----:            |  :---:          |    :----                                     |
| `config`   | **X** (if `Replication_info` is not empty)   | **X**           | JSON file       | Sets the PG connexion string (superuser)     |
| `patroni`  | **X**              |                    | JSON file       | Sets parameters for this special mode        |

## Session parameters template file creation

| Name                    |  Mandatory         | Optional           | Value expected   | Description                       |
| :---                    |    :-----:         |  :----:            |  :---:           |    :----                          |
| `config`                | **X**              |                    | JSON file        | Sets the PG connexion string      |
| `create_gucs_template`  | **X**              |                    | output name file | Sets the template file to create  |

# Reference: parameters and flags

All flags and parameters can be listed executing `pgSimload -h`.

There are 2 different modes when executing `pgSimload`:
  
  - **SQL-Loop mode** to execute a script infintely on a given schema of a
    given database
  - **Patroni-Watcher mode** to execute a monitoring on a given Patroni
    cluster. So you need one of such for this mode to be useful to you

Given the mode you choose, some parameters are mandatory or not. And the
contexts of executions are different.

Before listing each, there are common parameters that can be used. Let's see
those first.

## Common flags and parameters

### **config** (JSON file) [MANDATORY]

In the **SQL-Loop mode** the "Username" set in the `config.json` can be any
PostgreSQL user.

In the **Patroni-Watcher mode** the "Username" set in the `config.json`
**has to be a superuser** in PostgreSQL, typically "postgres". Because we use
special tricks to get the `hostname` of the PostgreSQL primary server.

"ApplicationName" is used to put a special "pgSimload" there, so the user can
`ps aux | grep [p]gSimload` on any of the PostgreSQL server to isolate the
process pgSimload uses... Or for any other SQL / bash command.

A valid `config.json` looks like this:

```code
{
   "Hostname":         "localhost",
   "Port"    :         "5432",
   "Database":         "mydbname",
   "Username":         "myusername",
   "Password":         "123456",
   "Sslmode" :         "disable",
   "ApplicationName" : "pgSimload"
}
```

"Sslmode" has to be one among those described in [Table 34.1. SSL Mode
Descriptions](https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-PROTECTION).

Most common values would be there either `disable` for non-SSL connexion or
`require` for SSL ones.

### **contact** (flag) [OPTIONAL]

Executing with only `-contact` will show you where you can contact the
programmer of the tool. 

This flag is not supposed to be run with other parameters or flags.

### **help** (flag) [OPTIONAL]

Originally, "heredocs" were used in the main program to show this help, but it
became too big to do such, it's better to have that doc in the current format
you're reading, makes the source code lighter and that's cleaner IMHO.

So as per now, the execution of that `-help` is only to show where the current
documentation is located. Actually, if you are reading this, that means wether
you executed that flag...or that you find it by yourself. Kudos :-)

This flag is not supposed to be run with other parameters or flags.

### **license** (flag) [OPTIONAL]

Executing with only `-license` will show you the license of this tool,
currently licensed under The PostgreSQL License.

A full copy of the licence should be present aside the tool, in the main
directory, in a file named `LICENCE.md`.

This flag is not supposed to be run with other parameters or flags.

### **silent** (flag) [OPTIONAL]

Executing pgSimload in any mode with that `-silent` flag added to the command
line will remove the welcome banner of pgSimload when you execute it. That
banner shows the copyright, name and version of the program, the mode you're
currently using and a reminder to exit the program.

When executing pgSimload in the `-silent` mode, please remember that you can
exit pgSimload anytime by pressing the `ESC` key.

### **version** (flag) [OPTIONAL]

Executing with only `-version` will show you the current version of pgSimload.
This is intended for general information of the users and also for any further
packager of the tool in various systems. 

Not supposed to be run with other parameters. No need to add a value to that flag.

## SQL-Loop mode parameters

The `config` flag is not listed down there, but is still **mandatory** to run
in this mode, please read carefully informations upper in this documentation.
On this mode, no particular "Username" has to be set in the `config.json`
file.

### **clients** (Integer) [OPTIONAL]

This parameter has been added in version 1.4.0. 

It allows to specify a number of *parallel executions* of the main SQL-Loop.

There's no verification or calculations whatsoever of available free
connections (like testing the values of `max_connections`,
`superuser_reserved_connections`, active connections, etc..). So if you hit a
limit here in terms of connections, you'll have an error indicating exactly
that. So that's up to you to put a decent number here... If you need (many)
hundreds or connections, I strongly encourage you to use a pooler like
[PgBouncer](https://www.pgbouncer.org/).

If you use limitations (`-loops` and/or `-time` and/or `-sleep`) and/or
special session parameters (`-session_parameters`), those will be applied to
all clients the same way.

Note that if you do test also HA with this parameter enabled, the output on
the console about disconnections, errors, reconnections, etc... Will be
multiplied by the number of clients you're using.

If you want to track number of pgSimload connections on your database, you
could use queries like:

```
select pid, state 
from pg_stat_activity 
where application_name = 'pgSimload';

select application_name,count(*) 
from pg_stat_activity 
where application_name = 'pgSimload' 
group by 1;
```

And you can simulate problems in the database in many ways: stopping the
PostgreSQL cluster, deleting a PostgreSQL cluster's files (if you're using
**and testing** HA...), or simply terminating PostgreSQL backends as super
user like this:

```
select pg_terminate_backend(pid) 
from pg_stat_activity 
where application_name = 'pgSimload';
```

If you do this, you should see pgSimload attempting to reconnect, and drop
trying that after 1 minute of failure(s).

### **create** (JSON text file) [OPTIONAL]

If you need to create tables, or do anything prior to the execution of the
main loop, you have to put your SQL commands in this JSON text file.

This script will be run only once prior the main loop on the `script`
described above.

If you're want to execute pgSimload in SQL-Loop mode on an existing database,
on which you've adapted the SQL present in the `script`, then you don't need
this feature. That's why it is optional.

To have a better idea of what's expected here, please refer to
`examples/SQL-Loop/PG_15_Merge_command/create.json` or
`examples/SQL-Loop/testdb/create.json` files.

### **script** (SQL text file) [MANDATORY]

This file is in plain text and contains SQL statements to run, in the main
loop of pgSimpload in the "SQL-Loop mode".         

It can be as simple as a "SELECT 1;". Or much more complex with SQL SQL
statements of your choice separated by semi-colon and newlines. As an example
of a more complicated example see `examples/SQL-Loop/PG_15_Merge_command/script.sql`.

It will run the querie(s) "all at once" in an implicit transaction. For more
details on how it works, please read chapter
[Multiple Statements in a Simple
Query](https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-MULTI-STATEMENT)
in the PostgreSQL's documentation.

### **session_parameters** (JSON text file) [OPTIONAL]

This parameter lets you tweak the PostgreSQL configuration that can be
specified in a session. This can be everything your PostgreSQL version
allows, and we let you define proper values for proper parameters.

Every parameter you specify here will be passed at the beginning of the
session when the SQL-Loop is executed. So everything will be executed
accordingly to those parameters in that session.

As an example, you can tweak `work_mem` in a session, or `synchronous_commit`,
depending your PostgreSQL configuration and version.

The format of the JSON file has to be the following:

```code
{
  "sessionparameters": [
    {
      "parameter" : "synchronous_commit"
     ,"value"     : "remote_apply"
    },
    { "parameter" : "work_mem"
     ,"value"     : "12MB"
    }
  ]
}
```

You can add as many parameters you want in that file, from one to many.

At the moment, we don't check if the parameter and values are OK. As an
example, if you set a value for an unknown parameter, you will have this
output when running pgSimload:

```code
The following Session Parameters are set:
   SET synchronous_commit TO 'remote_apply';
   SET work_mem TO '12MB';
   SET connections TO 'on';

2023/06/27 14:24:38 ERROR: unrecognized configuration parameter "connections" (SQLSTATE 42704)
```

Or if you set a right name, but in the wrong context you could have this too:

```code
The following Session Parameters are set:
   SET synchronous_commit TO 'remote_apply';
   SET work_mem TO '12MB';
   SET log_connections TO 'on';

2023/06/27 14:41:20 ERROR: parameter "log_connections" cannot be set after connection start (SQLSTATE 55P02)
```

And finaly, if you think you set proper values but it seems that nothing is
read from the brand new `session_parameters.json` you just created, like in:

```code
The following Session Parameters are set:


Now entering the main loop, executing script "./examples/SQL-Loop/testdb/script.sql"
Script executions succeeded :        60
```

... that's because you have probably a error in the JSON file, or maybe you
changed the keyword `"sessionparameters":` : don't do that, it's expected in
pgSimload to have such keyword there. It is also expected that your JSON file
here is valid, like given in the example file given in 
`examples/SQL-Loop/testdb/session_parameters.json`

### **loops** (integer64) [OPTIONAL]

This parameter was added in version 1.1.0. 

It allows one to limit the number of times the SQL-Loop will be run.

As an example, passing the `-loops 10` parameter to your `pgSimload` command
line will give you the following (extract of) output:

```code
[...]
Now entering the main loop, executing script "light.sql"

Number of loops will be limited:
    10 executions
Script executions succeeded :         10                               
=========================================================================
Summary
=========================================================================
Script executions succeeded :         10 (9.561 scripts/second)
Total exec time             :     1.045s
=========================================================================
```

This parameter can be used in conjuction with the `-time` parameter below.
Whichever is satisfied first will end the SQL-Loop.

### **time** (duration) [OPTIONAL]

This parameter was added in version 1.1.0. 

It allows one to limit the execution time of the SQL-Loop.

The value has to be one of "duration", that is expressed with or without
simple or double quotes, so all the following values are valid:

  - 10s for ten seconds
  - "1m30s" for one minute and thirty seconds
  - '1h15m4s" for one hour fifteen minutes and four seconds

Note that GoLang's Time package limits duration units to the following list,
you can use to pass the duration you want:

  - "ns" for nanoseconds 
  - "us" (or "µs") for microseconds
  - "ms" for milliseconds
  - "s" for seconds
  - "m" for minutes
  - "h" for hours

I hardly can believe you'd ever want pgSimload to run for days, months, year..
Don't you?

As an example passing the `-time 10s` parameter to your `pgSimload` command
line will give you the following (extract of) output:

```code
Now entering the main loop, executing script "light.sql"

Number of loops will be limited:
    "10s" maximum duration
Script executions succeeded :         90                               
=========================================================================
Summary
=========================================================================
Script executions succeeded :         90 (8.907 scripts/second)
Total exec time             :    10.104s
=========================================================================
```

This parameter can be used in conjuction with the `-loops` parameter seen in
the previous paragraph. Whichever is satisfied first will end the SQL-Loop.

### **sleep** (duration) [OPTIONAL]

This parameter was added in version 1.2.0.

It allows the user to actually throttle down the execution in SQL-Loop mode.

A pause of the `sleep` duration will be added between each iteration of the
execution of the SQL script.

That duration is expressed the very same way the `time` parameter is (see
previous paragraph). 

Note that when `sleep` and `time` are used together, it can cause side effects
on the total desired execution time, as an example:

  - the script is a plain "select 1;" that goes ultra fast (time to exec is 
    close to 0 seconds)
  - `time` is set to `10s`
  - `sleep`  is set to `4s`
  - the total execution time *won't be* 10s, but rather 12s, because on the 
    3rd execution at 8s, the pause will last 4s, leading to 12s overall... 

Just bare that in mind when creating your tests use cases, etc.

## Kube-Watcher mode flag and parameters

To have pgSimload act as a Kube-Watcher tool inside a termina, all you have to
do is create a `kube.json` file in the following format.

### **kube** (JSON text file) [MANDATORY]

When this parameter is set (`-kube <kube.json>`), you're asking pgSimload to
run in Kube-Watcher mode. This mode means that you want pgSimload to monitor
your PostgreSQL cluster that runs in Kubernetes, whether this one uses *or
not* Patroni or whatever for HA, or even if the cluster is *not* in HA. 

Simply, this mode runs some `kubectl` commands in the background, so it is
**mandatory** you have a working `kubectl` command installed in your system.

This command will show relevant info on your pods: 

  - which is primary
  - which is(are) replica(s) 
  - the status of each pod
  - a colored button in that description, so
    - green is a primary
    - blue is a replica
    - red means the pod is down for some reason

So that information won't show the TimeLine (aka TL), or Lag between nodes,
unlike Patroni-Watcher mode would do. But still, it's probably sufficient in
many cases, and is a light-weight monitoring of a PostgreSQL cluster in
Kubernetes!

I've added 2 samples JSON configurations you may even not have to change:
 
  - `examples/Kube-Watcher/kube.json.PGO` is the one to be used for 
    any Crunchy Postgres for Kubernetes and/or Postgres Operator from 
    Crunchy Data
  
  - `examples/Kube-Watcher/kube.json.CloudNativePG` is the one to be used
    for any CloudNativePG PostgreSQL cluster

Let's present that JSON with the PGO version: only the values will differ.

An example file is given in `examples/Kube-Watcher/kube.json.PGO`:

```code
{
    "Namespace"        : "postgres-operator", 
    "Watch_timer"      : 2,
    "Limiter_instance" : "postgres-operator.crunchydata.com/instance",
    "Pod_name"         : "NAME:.metadata.name",
    "Pod_role"         : "ROLE:.metadata.labels.postgres-operator\\.crunchydata\\.com/role",
    "Cluster_name"     : "CLUSTER:.metadata.labels.postgres-operator\\.crunchydata\\.com/cluster",
    "Node_name"        : "NODE:.spec.nodeName",
    "Pod_zone"         : "ZONE:.metadata.labels.topology\\.kubernetes\\.io/zone",
    "Pod_status"       : "STATUS:status.conditions[?(@.status=='True')].type",
    "Master_caption"   : "leader",
    "Replica_caption"  : "... replica",
    "Down_caption"     : "<down>"
}
```

Beware those `\\.` (double-escaping) are not errors, that's the way to specify
things in `kubectl` : those "dots" have to be escaped. And in pgSimload, since
it's a JSON, it has to be escaped *twice*...

**Namespace** (string)

To specify on which kube namespace we will send the underlying `kubectl`
commands.

**Watch_timer** (integer)

How often to refresh the output you see in this Kube-Watcher mode.

**Limiter_instance** (string)

From that field in the JSON, up to **Master_caption**, you won't probably have
to change those, if you're using the *right* example JSON given in the
directory `examples/Kube-Watcher/`.

This parameter is a way to limit the output of `kubectl get pods` commands
("-l").

**Pod_name** (string)

This is how we can get the pod's name.

**Pod_role** (string)

This is the way we get the pod's role, that can be:

  - a "replica" for PGO and CloudNativePG

  - a "master" (for PGO) or a "primary" (for CloudNativePG)

**Cluster_name** (string)

This is the way we get the PostgreSQL's cluster name.

**Node_name** (string)

This is the way we get node node's names. That Node name is used in the 2nd
`kubectl` command that is a `kubectl get nodes` one, to get back the Zone and
the Status of the given node.

**Pod_zone** (string)

This specifies how to gather the pod's zone, if it exists.

**Pod_status** (string)

This specifies how to gather the pod's status.

**Master_caption** (string)

This parameter allows the user to set the caption that will be displayed in
the Kube-Watcher mode for the primary Postgres instance in the cluster.

It can be whatever you like. Most probably, "primary" and "leader" are good
candidates here!

**Replica_caption** (string)

This parameter allows the user to set the caption that will be displayed in
the Kube-Watcher mode for all the instances that are actually replicas of the
primary.  

It can be whatever you like. Most probably, "replica" is great enough... 
"..replica" is good too, if you want to show some hierarchy in between
replica(s) and the primary.

**Down_caption** (string)

Finaly, this paramter allows you to specify the caption for any "down" pod.

## Patroni-Watcher mode flag and parameters

To have pgSimload act as a Patroni-Watcher tool in a side terminal, all
you have to do is to create a `patroni.json` file in the following format.
Note that the name doesn't matter much, you can name the way you want.

### **patroni** (JSON text file) [MANDATORY]

When this paramter is set (`-patroni <patroni.json>`), you're asking pgSimload
to run in Patroni-Watcher mode. This parameter is used to give to the tool
the relative or complete path to a JSON file formated like the following
example you can find a copy with the file
`examples/Patroni-Watcher/patroni.json`:

```code 
{
    "Cluster"          : "mycluster", 
    "Remote_host"      : "u20-pg1",
    "Remote_user"      : "postgres",
    "Remote_port"      : 22,
    "Use_sudo"         : "no",
    "Ssh_private_key"  : "/home/jpargudo/.ssh/id_patroni",
    "Replication_info" : "server_version,synchronous_standby_names,synchronous_commit,work_mem",
    "Watch_timer"      : 5,
    "Format"           : "list",
    "K8s_selector"     : "",
    "K8s_namespace"    : ""
}
```

**Cluster** (string)

You must specify here the Patroni's clustername. You can generaly find it
where your have Patroni installed in `/etc/patroni/<cluster_name>.yml` or
inside the `postgresql.yml`. 

**Remote_host** (string)

You have to set here the `ip` (or `hostname`) where pgSimload will `ssh` to
issue the remote command `patronictl` as user `patroni_user` (see up there).

**Remote_user** (string)

This is an user on one of the PG boxes where Patroni is installed. That one
you use to launch Patroni's `patronictl`. Depending the security configuration
of your PostgreSQL box, Patroni could run with the system account PostgreSQL
is running with, or another user. This one may have need to use `sudo` or not.
Again,that all depends on your setup.

**Remote_port** (integer)

You have to set here the `port` on wich  pgSimload will `ssh` to. Let the
default `22` if you didn't changed the `sshd` port of your remote server.

**Use_sudo** (string)

If the previous user set in **Remote_user** needs to use `sudo` before issuing
the `patronictl` command, then set this value to "yes".

**Ssh_private_key** (string)

Since `ssh`-ing to the `Remote_host` IP or (`hostname` if it's enabled in your
DNS) need an SSH pair of keys to connect, we're asking where is that private
key. It can be as simple as `/home/youruser/.ssh/id_patroni`.  Beware not to
set here the public key, because we need the private one.

Also, we assume you did the necessary thing on SSH so that user can SSH from
the box where pgSimload is running to the target host, specifically, that the
public key of your user is present in the `~/.ssh/authorized_keys` of the
taget system and with the matching **Remote_user**.

**Replication_info** (string)

Thanks to this feature, pgSimload can show extra information about
replication. This is usefull if Patroni doesn't do "everything in HA", like
the SYNChronous replication, that can be handled by PostgreSQL itself, thanks
to the `synchronous_commit` and `synchronous_standby_names` parameters. It can
also adapt in other scenarios, or just to show the `server_version`, whatever
you want!

If you don't need this extra information, to disable it, just set it to an
empty string in the JSON like: 

```code
[...]
    "Replication_info" : "",
[...]
```

If disabled, the "Replication information" no extra information will be shown
after the output of the `patronictl ... list` command.

If you want to activate it, like `Replication_info` is anything different to
an empty string, **be sure you also provide also `-config <config.json>`
parameter, pointing to a file where superuser `postgres` connection string is
defined**. So that in this config file, "Username" should be set to "postgres",
and the PG box name and port should be directly set.

So there's 2 ways to activate this feature described above.

If you want to activate it, but want pgSimload to only show othe output some
extra information from `pg_stat_replication` system table, then you set the
special value "nogucs" like:

```code
[...]
    "Replication_info" : "nogucs",
[...]
```

The other way to activate it is to ask pgSimload to show also settings from
the PostgreSQL Primary the whole query will be sent to. In this case, you have
to set there all the GUCs you want to be shown, you just have to name those
settings separated by a comma in the value of that JSON's fied.

This can be something like:

```code
[...]
    "Replication_info" : "synchronous_commit,server_version,work_mem,synchronous_commit",
[...]
```

You can look at examples given in `examples/Patroni-Watcher/` README and
subdirectories.

**Watch_timer** (integer)

You can ask for the output in the Patroni-Watcher mode to be like a bash
"watch" command: it will run every `x` seconds you define here.

If you want the tool to issue commands each 5 seconds, then set this parameter
to simply `5`. Since `patronictl` command can take several seconds to
run, the value you set here will be computed by the program to match your
request, with timers to take into account the time of execution. So then the
tool will iterate a bit before going the closest possible to your match your
request.

If the value is less than 1, pgSimload will assume you only want to run it
once in the Patroni-Watcher mode.

**Format** (string)

The `patronictl` command offers two modes to list the nodes:

  - `list` will order nodes output by name while
   
  - `topology` will show the Primary first, so the order may change if 
    you do a *switchover* of a *failover* 

**K8s_selector** (string)

This parameter has to be set **only if your PostgreSQL Patroni cluster is in
Kubernetes**.

The value of this field must be what you'd put in the "selector" chain of that
particular `kubectl` command, if you want to get the name of the pod where the
current PostgreSQL primary is executing into, and you're running Crunchy
Postgres for Kubernetes, that could be done with the following command:

```code
kubectl get pods \
  -n postgres-operator \
  --selector='postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master' \
  -o name 
```

So pgSimload knows the pod where the Primary PostgreSQL server is running.

The usage of pgSimload in Patroni-Watcher mode in Kubernetes **has
requirements**, we urge you to read carrefully the documentation you can
access at `examples/Patroni-Watcher/README.md` !

In short, if the Patroni-Watcher mode has to be executed on a cluster of
PostgreSQL servers in Patroni, the only relevant paramters in the patroni.json
file would then be:

  - `Replication_info` : can be set to an empty string (`""`), if you dont
    need it, `nogucs` or `<list of GUCs separated by a coma>` if you want
    those informations to be shown. In the later case, you'll need then to run
    mandatorily with the `-config config.json` parameter too. In 
    than file you'll set a superuser connection (e.g. "postgres" username)
  - `Watch_timer` has to be set to a value >1 otherwise it will only runs once
  - `Format` has to be set either to `list` or `topology`. In `list`, nodes
    will be ordered by name, while in `topology`, the Primary will be shown 
    first
  - `K8s_selector` has we already seen up there
  - all others parameters won't apply, so you can leave them empty ("")

**K8s_namespace** (string)

This parameter has to be set **only if your PostgreSQL Patroni cluster is in
Kubernetes**.

You will set this value to the namespace you're using for your current
PostgreSQL Cluster.

If you're using Cruchy Postgres for Kubernetes, and you're actually following
the documentation in there, that could be `postgres-operator`, or anything
you're actually using.

## Session parameters template file creation

### **config** (value) [MANDATORY]

Same as before, you define in that `config.json` file (or whatever the name,
but it has to be a valid JSON here: see previous examples) the connection that
will be used to query the
[pg_settings](https://www.postgresql.org/docs/current/view-pg-settings.html) system view.

You can use whatever user here (i.e. superuser or not), because we only gather
the parameters in the `user` context as per
[pg_settings](https://www.postgresql.org/docs/current/view-pg-settings.html)
PostgreSQL documentation.

### **create_gucs_template** (value) [MANDATORY]

This parameter should have been named
`create_session_parameters_template_file` to understand what it does... 

Here, pgSimload will connect to a given PostgreSQL server as described in the
mandatory  `-config <config.json>` parameter you have to use too. Then, it
will query the system view
[pg_settings](https://www.postgresql.org/docs/current/view-pg-settings.html)
to gather the name and the value (aka `setting`) of each parameter than can be
changed in a given session.

Then it will output that in file which format is expected by pgSimload to be
passed to the parameter `-session_parameter`.

Beware that those parameters change from one major PostgreSQL version to
another, so likely a file you previously generated, then edited to suit your
needs, on a version 15 won't work on a version 12. 

Also, since ALL parameters in the context `user` will be gather (see
[pg_settings](https://www.postgresql.org/docs/current/view-pg-settings.html)
for details), there will be likely many dozens of parameters here. As an
example, as per version 15, it's more than 130 parameters...

Since you probably won't need all of these, most likely, you run that command
once to have every parameter in the generated template, then you edit it to
remove all uncessary parameters. You'll have then your own template you can
use in different scenarios, creating as many `session_parameters.json` you
need, to be tested.

