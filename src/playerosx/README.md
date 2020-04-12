# 4klang rendering example for Mac OSX

In here you'll find `4klangrender.c` which will call the `4klang.asm` library and render one tick
at the time (using the `AUTHORING` flag in `4klang.inc`).

The `runosx.sh` shell script contains the commands to compile the asm and link with the c example code, and create
an executable that will ouput a stream of 32 bit floating point samples. By piping into [SOX](http://sox.sourceforge.net/) we'll get sound output.

You also need [YASM](http://yasm.tortall.net/) and gcc (XCode) installed, and if everything is right you should get sound output by simply typing in your terminal (from this folder):

`sh runosx.sh`