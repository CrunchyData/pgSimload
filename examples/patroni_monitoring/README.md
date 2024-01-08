# Patroni monitoring examples

The "Patroni-monitoring mode" requires at least one configuration file, the one
you pass in the parameter `-patroni <patroni.json>` file.

If you want special extra informations taken mostly from `pg_stat_replication`
table, then you'll need also a configuration file for the connexion to the
database. This connexion has to be executed with a superuser in PostgreSQL,
aka `postgres` user (or any superuser in PostgreSQL you defined).


## Usage in localhost/baremetal/VMs...

### Patroni config file

When you use your cluster in baremetal or VMs, you should have every parameter
in the `<patroni.json>` **except** the `K8s_selector` that **must remain
empty**.

This could be something like:

```code
{
    "Cluster"          : "Patroni_cluster_name", 
    "Remote_host"      : "Primary_hostname_or_IP",
    "Remote_port"      : 22,
    "Remote_user"      : "username_on_the_remote_host",
    "Use_sudo"         : "yes",
    "Ssh_private_key"  : "/home/your_username_here/.ssh/id_patroni",
    "Replication_info" : "synchronous_commit,server_version,work_mem,synchronous_commit", 
    "Watch_timer"      : 5,
    "Format"           : "list",
    "K8s_selector"     : ""
}
```

`Cluster` is the name of the Patroni cluster itself you want to monitor.

`Remote_host` and `Remote_port` are used to connect the `Remote_user` via SSH.
If you use some different port for `sshd` on the `Remote_host`, then please
change the port from 22 to that port.

If the `Remote_user` set has to use `sudo` command to run `patronictl` then,
please set `Use_sudo` to `yes`, ortherwise, to `no`. If `sudo` is not used,
then it's assumed that `Remote_user` has access and rights to use
`patronictl`...

`Replication_info` is a versatile one. It accepts 3 kinds of parameters:

  - "" : an empty string, so you set `"Replication_info" : "",` (don't forget
    the "," at the end of the line otherwise your JSON is not valid. If you 
    set this, the "Replication information" output is limited to what's in 
    `pg_stat_replication` and no GUCS (from `pg_settings`) are shown

 - "nogucs": this is a very special value, so you set `"Replication_info" :
   "nogucs",` at this line in the JSON file. In this case, there's no "added
   GUCS" shown in the output of "Replication information". Only the info from
   `pg_stat_replication` is shown 

  - "guc1, guc2 .. , guc[n]" : you can also set a list of gucs you want to 
    be shown as extra information added to the things from `pg_stat_replication` 
    system table. This can be something like : `"Replication_info" :
   "work_mem, synchronous_commit, synchronous_standby_names",`. All "names"
   (aka `name` column in `pg_settings` system table) and their actual values
   will be added to the output of "Replication information"

`Watch_timer` is like the `-n <seconds>` you have with `watch -n <seconds>`.
The information will refresh every `n` seconds.

`Format` is the way you want Patroni to show you the information:

  - `list` will show the nodes ordered by name while

  - `topology` will show you the nodes ordered by role, where the Leader will
    be shown first

It can be only one or the other.

The `K8s_selector` has to be empty (i.e. let it to `""`), if you're running in
baremetal or VMs.

### Connexion info

Only if you set `Replication_info` to `on`, then you need to pass also the
`-config <config.json>` parameter to pgSimload aside the `-patroni...` one.

Because we use special tricks in PostgreSQL to gather hostnames, a superuser
connexion has to be used.

So the `config.json` should then be like:

```code
{
    "Hostname":         "cluster_vip",
    "Port"    :         "5432",
    "Database":         "postgres",
    "Username":         "postgres",
    "Password":         "verysecret",
    "Sslmode" :         "disable",
    "ApplicationName" : "pgSimload"
}
```

If you're using like KeepAliveD to access the whole cluster and HAProxy to
point out to the read/write port, then this example could be OK.

You can also point out directly to your PostgreSQL server, depends on what you
want to do!


## Usage in Kubernetes

The Patroni monitoring can also work in Kubernetes !

There are **requirements** for that special usage to run OK:

  - you have a working `kubectl` on the box where pgSimload is running

  - the currently namespace you work on is `postgres-operator`

You'll need to give the right "selector" in the `patroni.json` file so
pgSimload knows wich of your pods contains the Primary PostgreSQL server.

Like in :

```code
$ kubectl get pods --selector='postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master' -o name

pod/hippo-instance1-mr6g-0
```

So the `patroni.json` will be then very different in terms of values:

```code
{
    "Cluster"          : "", 
    "Remote_host"      : "",
    "Remote_port"      : 22,
    "Remote_user"      : "",
    "Use_sudo"         : "",
    "Ssh_private_key"  : "",
    "Replication_info" : "synchronous_standby_names,server_version,synchronous_commit,work_mem",
    "Watch_timer"      : 5,
    "Format"           : "topology",
    "K8s_selector"     : "postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master"
}
```

You must then give the right "selector" in the parameter `K8s_selector` so
pgSimload will initiate proper `kubectl` commands to get Patroni information
from the Kubernetes cluter.

`Replication_info` can be `on` or `off`, depending your needs, if you want
more info from `pg_stat_replication`.

Again, `Format` can be `list` or `topology`, depending on want you want to be
shown 1st in the output of `patronictl`.

### Connexion info

Only if you set `Replication_info` to `on`, then you need to pass also the
`-config <config.json>` parameter to pgSimload aside the `-patroni...` one.

Because we use special tricks in PostgreSQL to gather hostnames, a superuser
connexion has to be used.

There are requirements also here to have things working: 

  - you did the necessary in your PostgreSQL cluster for the "postgres" user
    to be used in your cluster: see [Managing the postgres User](https://access.crunchydata.com/documentation/postgres-operator/latest/tutorial/user-management/)

  - you created the necessary for outside connexion to be possible on an
    EXTERNAL-IP of a Loadbalancer to your database (`kubectl get -n postgres-operator services` should list that for you)

  - you know how to retrive "postgres" password, if not you could patch it
    like in `kubectl patch secret -n postgres-operator hippo-pguser-postgres -p '{"stringData":{"password":"verysecret","verifier":""}}'`

So the `config.json` should then be like:

```code
{
    "Hostname":         "192.168.2.200",
    "Port"    :         "5432",
    "Database":         "hippo",
    "Username":         "postgres",
    "Password":         "verysecret",
    "Sslmode" :         "require",
    "ApplicationName" : "pgSimload"
}
```
