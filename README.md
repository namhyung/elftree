# ELF tree

Show library dependency of an ELF binary.

    $ elftree `which firefox`
    firefox
       libpthread.so.0
          libc.so.6
             ld-linux-x86-64.so.2
          ld-linux-x86-64.so.2
       libdl.so.2
          libc.so.6
          ld-linux-x86-64.so.2
       libstdc++.so.6
          libm.so.6
             libc.so.6
             ld-linux-x86-64.so.2
          libc.so.6
          ld-linux-x86-64.so.2
          libgcc_s.so.1
             libc.so.6
       libm.so.6
       libgcc_s.so.1
       libc.so.6
       ld-linux-x86-64.so.2
