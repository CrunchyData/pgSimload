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
8e2fef8861c61d81dcfa08f18d5c5e4c  pgSimload
eed6894cb131464e1b14c209d0f6236f  pgSimload_mac
8c80e9f2da22ad9a426945f54ec9019b  pgSimload_static
2c3016e3f89314dd950dcb82a13a4f27  pgSimload_win.exe
```

# More information

If you're a Mac (darwin/amd64) user or a Windows (windows/amd64) user it would
be straightforward: you just have to get the binary and use it, make it
executable if it's not already on your system.

If you're a Linux user, let me warn you that the binary has dependencies:
 
```
$ ldd pgSimload
	linux-vdso.so.1 (0x00007ffe7ccc8000)
	libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x0000790969200000)
	/lib64/ld-linux-x86-64.so.2 (0x0000790969614000)
```

Those depedencies would be OK if as an example you're using an up-to-date
Ubuntu 24.04 LTS Noble Numbat as per August, 28th 2024.

But for any other machine without those versions of dependencies, please
download and use instead the binary compiled *statically* for your convenience,
in the name of `pgSimload_static` !

Once downloaded, you can rename it the way you want like `pgSimload` or
`pgs`... and put it somewhere accessible to your `$PATH`, generally,
`/usr/bin/local/` or `~/bin/` are good canditates for that.
