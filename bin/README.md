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
7401abb6f3fe7bec2260c8a97885680d  pgSimload
f37240073b23dc4d6a84b8e641ecbdb4  pgSimload_mac
e695d36afd36781d074f1517ebe3a185  pgSimload_static
13035fd9a0416f2a113c8b166890143e  pgSimload_win.exe
```

# More information

If you're a Mac (darwin/amd64) user or a Windows (windows/amd64) user it would
be straightforward: you just have to get the binary and use it, make it
executable if it's not already on your system.

If you're a Linux user, let me warn you that the binary has dependencies:
 
```
$ ldd pgSimload
	linux-vdso.so.1 (0x00007ffe91dc8000)
	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x0000744b2d600000)
	/lib64/ld-linux-x86-64.so.2 (0x0000744b2d90b000)
```

Those depedencies would be OK if as an example you're using an up-to-date
Ubuntu 24.04 LTS Noble Numbat as per July 25th 2024.

But for any other machine without those versions of dependencies, please
download and use instead the binary compiled *statically* for your convenience,
in the name of `pgSimload_static` !

Once downloaded, you can rename it the way you want like `pgSimload` or
`pgs`... and put it somewhere accessible to your `$PATH`, generally,
`/usr/bin/local/` or `~/bin/` are good canditates for that.
