echo "Copying 4klang.asm and 4klang.inc from parent dir"
echo "Enabling SINGLE_TICK_RENDERING (and AUTHORING) flag for rendering example"
echo "You need sox installed to get audio output"
sed -e 's/\;\%define SINGLE_TICK_RENDERING/\%define SINGLE_TICK_RENDERING/' ../4klang.inc | \
    sed -e 's/\;\%define AUTHORING/\%define AUTHORING/' > 4klang.inc
cp ../4klang.asm .
yasm -f macho 4klang.asm
gcc -Wl,-no_pie -m32 4klang.o 4klangrender.c -o 4klangrender
./4klangrender | sox -t raw -b 32 -e float -r 44100 -c 2 - -d
