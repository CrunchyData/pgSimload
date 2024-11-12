# Release notes

## Version 1.4.2 (November, 12nd 2024)

### Major changes

- **none** this is mostly a monthly build, to refresh dependencies'versions

### Minor changes

- fixed a small typo in a `sqlloop.go` output

- removed `bin/` 
  - to lighten sources! 
  - binaries will be provided only within [the releases](https://github.com/CrunchyData/pgSimload/releases) starting from
today
  - reviewed a bit documentation to act that change

- rebuild of binaries with dependencies updates:
  - upgraded golang.org/x/crypto v0.28.0 => v0.29.0
  - upgraded golang.org/x/sys v0.26.0 => v0.27.0
  - upgraded golang.org/x/text v0.19.0 => v0.20.0

## Version 1.4.1 (October, 16th 2024)

### Major changes

- **Added a new parameter in command line: `-rsleep time.Duration`**
  Like `-sleep`, this parameter will add a sleep time between 2 iterations 
  (executions) of the `-script script.sql` (or whatever it's name).

  The difference is that the actual sleep time will be a random duration
  between 0 and that parameter. This is usefull when there are multiple 
  `-clients` running, so they don't fire each at the same exact time... 
  Or when one wants to execution to be just random.

  If used aside `-sleep time.Duration`, then the total sleep time between
  two iterations will be sum of both, one being a fixed duration and the other
  being a random one.

### Minor changes

- review of **examples** in the documentation, since many features have been
  added those last months, and the examples pages weren't updated accordingly.

## Version 1.4.0 (October, 11th 2024)

### Major changes

- **Added a new parameter in command line: `-clients <integer>`** 
  This parameter is only effective in the "SQL-Loop". It will create as many 
  connections to the database to execute a given script in parallel, creating 
  as many PostgreSQL connections as necessary, i.e. the number of clients
  declared here. There's *no* verification whatsoever of `max_connections` or 
  any computation of free connections. So if the user specifies too many 
  connections, an error message will be shown accordingly. If you need many
  (many) hundreds of concurrent connections, please consider the usage of 
  a PostgreSQL pooler, like [PgBouncer](https://www.pgbouncer.org/)
  
### Minor changes

- Support for Patroni v4 added, it was as simple as add the colorized output
  in `patronictl` for "Quorum Standby"

- raised the reconnection timeout from 20s to 60s (`pgReconnectTimeout`
  constant declared in `pgmanager.go`), because in certain scenarios 
  where there are a lot of `-clients ` used, the server may be in 
  heavy load enough so 20s isn't sufficient. I'm planning to make
  this usage parameter rather than a constant, to give control to users

- first usage of the "sync" library in `sqlloop.go`, to manage synchronicity
  among Go Routines used when the `-clients <integer>` new parameter is used

- rebuild of binaries with dependencies updates:
 
  - upgraded github.com/jackc/pgx/v5 v5.6.0 => v5.7.1
  - upgraded golang.org/x/crypto v0.26.0 => v0.28.0

## Version 1.3.3 (August, 28th 2024)

### Major changes

- **Added a new flag in command line: `-silent`**
  With that parameter added, pgSimload won't show anymore the welcome message
  (copyright, program name and version, execution mode, etc.). 
  The user will have to remember that the `ESC` key will still have to be
  pressed to get out of the execution loop

### Minor changes

- Adding `doc/build_doc.sh`, making it public

- Reviewed the way the documentation is produced with internal 
  script (`doc/build_doc.sh`). Had to change `main.go` to have `Version`
  and `Release_date` on separated variables

- Removed `$` in `$ command <exemples>` so the reader can copy/paste
  the code directly reading the documentation on github. So it is a 
  better user experience

- rebuild of binaries with dependencies updates:
 
  - upgraded golang.org/x/crypto v0.25.0 => v0.26.0
  - upgraded golang.org/x/sys v0.22.0 => v0.24.0
  - upgraded golang.org/x/text v0.16.0 => v0.17.0

## Version 1.3.2 (July 10th 2024)

### Major changes

None, this is a maintenance release.

### Minor changes

- Reviewed output of "Replication information" in the Patroni 
  Watcher mode, a bit less things, but, clearer and complete 
  presentation of what's important

- Fixed a bug in coloring patronictl (list|topology) command between
  "Leader" and "Standby Leader", now both are in red

- Fixed useless waiting loops in Patroni Watcher command, in the 
  Kubernetes flavor. It's now more robust to what could happen inside 
  the Kubernetes cluster. 

- Fixed caption in "Patronictl output from...", there was too many
  spaces in there

- re-Build of binaries to update underlying dependencies:
 
  - upgraded golang.org/x/crypto v0.24.0 => v0.25.0

  - upgraded golang.org/x/sys v0.21.0 => v0.22.0

## Version 1.3.1 (June 18th 2024)

### Major changes

None, this is a maintenance release.

### Minor changes

- Some captions revisited for better understanding by the user, and to be
  consistent. Particularly true for the "SQL-Loop" mode where the start 
  process was different

- Corrected some startup tests when mandatory files are not passed in the
  command line

- Corrected a bug when `kubectl` wasn't installed on default directories.
  This was preventing users to have installs like `/usr/local/bin/kubectl`.
  Now, `kubectl` can be anywhere in the `$PATH` of the user 

- re-Build of binaries to update underlying dependencies

## Version 1.3.0 (April 24th 2024)

### Major changes

- Added a new "Kube-Watcher" mode! This one allows to have a minimal 
  monitoring of any PostgreSQL cluster running in Kubernetes, whatever
  the operator is, since it *only* uses 2 `kubectl` commands, in a loop, 
  to create the monitoring. See examples in `examples/Kube-Watcher/`
  and the relevant parts in the documentation on how to use it !

  - TL,DR: simple as `pgSimload -kube <kube.json>`
  
  - Very special thanks to Brian Pace from Crunchy Data, since he
    shared with me his how script that was doing that, exactly,
    I found that it could be interesting to have it in pgSimload !

  - Special thanks to "Pierrick" from CloudNativePG Slack, that helped 
    me find the right labels and selectors for that operator to be used
    in the `kube.json` configuration file !

- documentation update to describe the new "Kube-Watcher" mode.

### Minor changes

- changed the way the screen is refreshed in Patroni-Watcher, for better
  performances and less output in the terminal 
  
  - moved from `github.com/inancgumus/screen`, `screen.Clear()` and
    `screen.MoveTopLeft()`
  
  - to simplied ANSI's `fmt.Printf("\x1bc")`

  - **please** let me know if this change break something for you, I can 
    revert that easily in case. I do really lack feedback from pgSimload
    users!

- simplier way of coding (most often, output messages) strings to be 
  used in output: `A+=B` instead of `A=A+B`. (/me noob)

- added 3 new functions to better padding (left/right) and count length of 
  some output strings (eg podname lenght in Kube-Watcher mode, to align
  the outputs when more than 1 cluster is listed in this mode)

- added `K8s_selector` field in `patroni.json` configuration file for 
  Patroni-Watcher, so it can be monitoring any namespace other than the 
  default set with, typically, `kubectl config set-context --current --namespace=<namespace_name>`

- renamed in doc, including examples directory and in executables'outputs
  everything to be consistent among the 3 modes: "SQL-Loop", "Patroni-Watcher"
  and the new "Kube-Watcher" in version 1.3.0

- lots of doc review, many errors find and corrected

## Version 1.2.0 (April 18th 2024)

### Major changes

- Added a new parameter to pgSimload command line to be used in SQL-Loop mode:

  - `-sleep time.Duration` adds a sleep time between 2 iterations (executions)
    of the `-script script.sql` (or whatever it's name).

  - The interest of this parameter is double:

    - it allows to throttle down the execution in SQL-Loop mode if this one is
      "going too fast" and
   
    - it avoids the user to add a line like `select pg_sleep(1);` at the end
      of the `script.sql`.

  - Actually it corrects indirectly the previous behaviour when that `select
    pg_sleep(n);` was used previously in `script.sql` around the count of
    statements executed. This one was only updated once the *whole* script was 
    executed, including the possible `select pg_sleep(n);` at the end.

- documentation update to describe the new `-loops` and `-time` parameters to
  be used in SQL-Loop mode

### Minor changes

- updated `examples/simple` examples and README file

### Minor changes

## Version 1.1.0 (January 20th 2024)

### Major changes

- Added 2 new parameters to pgSimload command line to be used in SQL-Loop
  mode:

  - `-loops <int64>` will limit the SQL-Loop execution to that exact number
    of loops. This can be used to avoid running SQL-Loop endlessly, and/or 
    in comparisons scenarios when one wants to compare effects of various 
    configurations parameters, including using different values when a 
    session parameters files is used (see `session_parameters <JSON.file>` 
    in docs) 

  - `-time time.Duration` (where Duration is a duration, without or with
    double or sigle quotes, like "10s" or 1m10s or '1h15m30s'...). This option
    will limit SQL-Loop execution to that amount of time. It can be used in 
    various scenarios too

  - when both are used at the same time, the SQL-Loop ends when any one of
    those conditions is satisfied

- documentation update to describe the new `-loops` and `-time` parameters to
  be used in SQL-Loop mode  

### Minor changes

- updated Crunchy copyright ranges to include 2024 (patch by @youattd)

- updated `examples/simple` examples and README file

- updated `examples/patroni_monitoring/README.md` doc to mention the

- added scripts in `examples/patroni_monitoring/ha_test_tools`

## Version 1.0.3 (January 15th 2024)

### Major changes

- In SQL Loop mode, don't ping the server in between operations, but only do
  that **on error** to check that the server is living or not. If not, try
  reconnecting as before... I used the right method for 1.0.2.. but at the wrong
  place. Sorry, this was eating useless performances! (ping roundtrip >> query
  exec in most scenarios..)

- don't parse `script.sql`. It's useless because Exec() handles multiple
  queries on a same file. And it does implicit transactions (so need to add
  begin/(commit|rollback) in the script file. Results in simplier code and
  fastest exection too!

### Minor changes

- `ioutils` usage replaced with `os`, because `ioutils` is deprecated. 
   So `ioutils` is removed everywhere too

- removed `Read_Config()` function in `main.go` : not used anymore

- review doc to state the major change around parsing/execution 

- fix rowcount == 0 in patroni.go / Replication info

- removed the test that looks for `ssh` binary locally, because what is is
  used is `sshManager.RunCommand(remote_command)`

- constructing the Replication info output in a string and throw it to the
  screen at once, rather than Println one line by one. To reduce flickering.

- added ComputedSleep() function in `patroni.go` to compute precisely how
  much to wait between 2 cycles tring to match user's expectations with
  `Watch_timer` parameter in the `patroni.json` file

- added a warning if the system takes longer to output than the user
  expects it to be with then `Watch_timer` parameter in the `patroni.json`
  file

- added a prior check in `sqlloop.go` to check the validity of the SQL
  in the `script.sql` file

- `strings` and `regexp` packages no more needed in `sqlloop.go`

- `github.com/jackc/pgx/v5` package no more needed in `patroni.go`

- changed "Statements" by "Scripts" in summary and loop info, because it's no
  more statements, it's the *whole* script that it is Exec() at once !

- review error code outputs

- pgReconnectTimeout moved from 30s to 20s, and moved to pgmanager.go (was in
  patroni.go)

- add more precise numbers in Summary of execution (times of
  execution/downtime and statements per second)

- corrected paragraph ordering in `doc/05_roadmap.md`

## Version 1.0.2 (January 11th 2024)

- split main.go in many other .go files for better
  maintenability. This will allow usage of Go Packages further more easily 

- split documentation in parts for better maintenability too

- main README.md of the project is a symlink to `/00_readme.md`

- README.md of the `doc/` is the same symlink

- new PGManager for everything

Same I did in version 1.0.1 with SSHManager, now PG connections are handled
by a manager. First, this bring cleaner code. Second, it allows pgSimload to
function with an unique connection to the PG database, wheter it is used in
SQL-Loop mode or Patroni-Watcher mode. It doesn't change dramatically things
in the SQL-Loop mode, because previously, an unique connection was used in the
main loop (but others to set transactions GUCS, if used, and Exectute script
if used, where still independant connections). But for the Patroni-Watcher, it
changes things a lot, allowing the Replication info output to be faster, and
offers less "flickering", because we don't pay anymore the connexion time,
which has the most cost in time execution.

- more code cleaning everywhere

## Version 1.0.1 (January, 8th 2024)

- new SSHManager for Patroni-Watcher mode

The way the Patroni-Watcher is handled in SSH (i.e not in Kubernetes modes)
has been refactored. Previously, an SSH connection was initiated at each loop
of the Patroni-Watcher. This was not very efficient, because at each
`Watch_timer` an SSH connection was opened, the `patronictl` command
initiated, the output shown, then the SSH connection was closed.

An SSHManager has then been added to manage this, at not only it is more
efficient, and an unique SSH connection is used, but also, it will manage any
disconnections of the SSH server itself, trying to reinitiate the SSH
connection if the previous died.

A bit more of code refactoring has been added too, so the dependances to the
`bytes` and `net` packages have been removed.

- new parameter in `patroni.json` file : `Remote_port` parameter has been
  added (integer), so you can specify the port of your SSH Server explicitery.
- updated Go modules
- rebuild of binaries
- tagging version 1.0.1
- updated any `patroni.json` file types in the examples to add `Remote_port`
- updated the documentation about `Remote_port` in `patroni.json` files

## Version 1.0.0 (December, 8th 2023)

After 3 months of intensive tests, pgSimload v.1.0.0 is out after the beta
period!

What's new?
 - updated Go modules
 - rebuild of binaries
 - tagging version 1.0.0
 - minor fixes in documentation (links)

## Version 1.0.0-beta (July, 24th 2023)

First release of pgSimload !

