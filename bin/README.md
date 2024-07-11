# Note on provided binaries 

You'll find here binaries for your convenience.

# Files

  - `pgSimload` : dynamically linked binary for `linux/amd64`
  - `pgSimload_static` : statically linked binary for `linux/amd64` if you
    have errors running the previous one, because of libraries mismatch (e.g.
    libc errors)
  - `pgSimload_mac` : binary to use in Mac `darwin/amd64`
  - `pgSimload_win.exe` : binary to use in Windows  `windows/amd64`

```
md5sum pgSimload*
b649c3cf2f4befb8e477572edee870ea  pgSimload
fdbc154af483a5567083b71ed1e628a4  pgSimload_mac
7ee01531cd6c3e69841c1aec58742ac5  pgSimload_static
0abeb72b685ad98edeb1e4cac3d6ffbf  pgSimload_win.exe
```

# More information

If you're a Mac (darwin/amd64) user or a Windows (windows/amd64) user it would
be straightforward: you just have to get the binary and use it, make it
executable if it's not already on your system.

If you're a Linux user, let me warn you that the binary has dependencies:
 
```
$ ldd bin/pgSimload
	linux-vdso.so.1 (0x00007fffa63f1000)
	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007791eac00000)
	/lib64/ld-linux-x86-64.so.2 (0x00007791eb01f000)
```

Those depedencies would be OK if as an example you're using an up-to-date
Ubuntu 24.04 LTS Noble Numbat as per June 18th 2024.

But for any other machine without those versions of dependencies, please
download and use instead the binary compiled *statically* for your convenience,
in the name of `pgSimload_static` !

Once downloaded, you can rename it the way you want like `pgSimload` or
`pgs`... and put it somewhere accessible to your `$PATH`, generally,
`/usr/bin/local/` or `~/bin/` are good canditates for that.
