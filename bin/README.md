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
$ md5sum pgSimload*
dc8d3d2be5039f96f64b37e706c348d0  pgSimload
fa409e0f8420cabe4205fe742c28ae6c  pgSimload_mac
e74b24db8ee92952a2988e7e99b18802  pgSimload_static
aedb0a8d1eebcec62fa0b3d6c9968186  pgSimload_win.exe
```

# More information

If you're a Mac (darwin/amd64) user or a Windows (windows/amd64) user it would
be straightforward: you just have to get the binary and use it, make it
executable if it's not already on your system.

If you're a Linux user, let me warn you that the binary has dependencies:
 
```
$ ldd bin/pgSimload
	linux-vdso.so.1 (0x00007ffd59dd7000)
	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007fa48aa00000)
	/lib64/ld-linux-x86-64.so.2 (0x00007fa48addb000)
```

Those depedencies would be OK if as an example you're using Ubuntu Mantic
Minautor (aka Ubuntu 23.10).

But for any other machine without those versions of dependencies, please
download and use instead the binary compiled *statically* for your convenience,
in the name of `pgSimload_static` !

Once downloaded, you can rename it the way you want like `pgSimload` or
`pgs`... and put it somewhere accessible to your `$PATH`, generally,
`/usr/bin/local/` or `~/bin/` are good canditates for that.
