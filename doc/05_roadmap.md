# Roadmap

## Short term

### Scenario mode

With PGManager tools I can now create the Scenario mode.

It will consist of running [1..n] SQL-Loop(s) at the same time on a server to
match real world usage scenarios.

A `scenario.json` will be created with:
  - ID of the client
  - associated caption to show on screen 
  - `config.json` associated file
  - `script.sql` to be used in the loop
  - `execution_number` parameter to configure how many times to execute the
    script (int64) (eg: 100)
  - `execution_time` parameter to configure how much time the Loop must be 
    run (time.Duration) (eg: "10m30s")
  - `output_type` : "none" or "eta"

The client will end its work wether if the `execution_number` *or* the
`execution_time` is satisfied. 

The execution in Scenario mode won't be interactive, except to be launched at
start, one will still have to press the Enter key to launch it.

The `output_type` will allow (as a start?) the user to set wheter: 
  - no output at all. The program will finish once every client is
    disconnected
  - a nice output on screen with colored progress bars, ETAs, etc (one line
    per client)

## More code cleaning

Because, heh, I'm a noob Go coder. Trying to do good, but I must admit it's a
long way.

## Study and pgmanager.go and jackc

I did this thing but I wonder if I'm using
[jackc](https://github.com/jackc/pgerrcode) properly... Probably what I've did
here is already done...

## Study and adapt pgmanager.go vs pgcon and pgerrcode

Same thing with pgcon I use for PG Error codes and
[pgerrcode](https://github.com/jackc/pgerrcode)... I trap some error codes to
output a right message to the user, but maybe I'd rather use this project,
that contains all the error codes PG has (v.14 still... hopefully PGDG won't
touch this ?)...


## Longer term

### Move to packages

Now every parts are in separated .go files I can think about building properly
independant packages to manage this. 
