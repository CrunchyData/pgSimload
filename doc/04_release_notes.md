# Release notes

## Version 1.0.2 (January 11th 2023)

- split main.go in many other .go files for better
  maintenability. This will allow usage of Go Packages further more easily 

- split documentation in parts for better maintenability too

- main README.md of the project is a symlink to `doc/01_readme.md`

- README.md of the `doc/` is the same symlink

- new PGManager for everything

Same I did in version 1.0.1 with SSHManager, now PG connections are handled
by a manager. First, this bring cleaner code. Second, it allows pgSimload to
function with an unique connection to the PG database, wheter it is used in
SQL-Loop mode or Patroni Watcher mode. It doesn't change dramatically things
in the SQL-Loop mode, because previously, an unique connection was used in the
main loop (but others to set transactions GUCS, if used, and Exectute script
if used, where still independant connections). But for the Patroni Watcher, it
changes things a lot, allowing the Replication info output to be faster, and
offers less "flickering", because we don't pay anymore the connexion time,
which has the most cost in time execution.

- more code cleaning everywhere

## Version 1.0.1 (January, 8th 2024)

- new SSHManager for Patroni Watched mode

The way the Patroni watcher is handled in SSH (i.e not in Kubernetes modes)
has been refactored. Previously, an SSH connection was initiated at each loop
of the Patroni watcher. This was not very efficient, because at each
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
