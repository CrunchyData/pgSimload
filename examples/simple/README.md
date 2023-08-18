# Most simple example

This example is the simpliest one:

Prerequisites

 - PostgreSQL Server, any version shoud work
 - user has LOGIN capabilities

No creation of tables and others are needed, so there's no need to call for
`-create <create.json>` or such. pgSimload accepts the omission of parameter
`-create`.

Here's something for the user to understand:

 - `script.one_liner.sql` contains a simple "select 1;" with a sleep(1) 
   **on the same line**, where

 - `script.two_liner.sql` contains the same, but each command on one line.

When you pass one or the other in the `-script <script.sql>` parameter, you'll
see the difference in counting "statements": actually, pgSimload counts
statements as one per line, because the parsing method for the `-script
<script.sql>` thing is very basic.

I did tests with a JSON version of it like `-script <script.json>` : it adds
an overhead and makes everything unreadable in the end. And pgSimload is not
designed to have "exact" parameters and results, the thing is having "some
data", and watch differences in between different configurations of PG
infrastructure, or parameters (GUCS), etc.

Once pgSimload has been compiled **and** the `config.json` adapted to suit
your needs, this could be used as simple as:

```code
$ pgSimload -config config.json -script script.one_liner.sql
$ pgSimload -config config.json -script script.two_liner.sql
```
