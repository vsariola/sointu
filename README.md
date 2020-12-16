# Sointu
![Tests](https://github.com/vsariola/sointu/workflows/Tests/badge.svg)

A cross-platform modular software synthesizer for small intros, forked from
[4klang](https://github.com/hzdgopher/4klang). Supports win32/win64/linux/mac.

Summary
-------

Sointu is work-in-progress. It is a fork and an evolution of
[4klang](https://github.com/hzdgopher/4klang), a modular software synthesizer
intended to easily produce music for 4k intros-small executables with a maximum
filesize of 4096 bytes containing realtime audio and visuals. Like 4klang, the
sound is produced by a virtual machine that executes small bytecode to produce
the audio; however, by now the internal virtual machine has been heavily
rewritten and extended. It is actually extended so much that you will never fit
all the features at the same time in a 4k intro, but a fairly capable synthesis
engine can already be fitted in 600 bytes (compressed), with another few hundred
bytes for the patch and pattern data.

Sointu consists of two core elements:
- A cross-platform synth-tracker app for composing music, written in
  [go](https://golang.org/). The app is not working yet, but a prototype is
  existing. The app exports (will export) the projects .yml files.
- A compiler, likewise written in go, which can be invoked from the command line
  to compile these .yml files into .asm code. The resulting single file .asm can
  be then compiled by [nasm](https://www.nasm.us/) or
  [yasm](https://yasm.tortall.net).

Building
--------

Requires [go](https://golang.org/), [CMake](https://cmake.org),
[nasm](https://www.nasm.us/) or [yasm](https://yasm.tortall.net), and your
favorite c-compiler & build tool. Results have been obtained using Visual Studio
2019, gcc&make on linux, MinGW&mingw32-make, and ninja&AppleClang.

### Example: building and running CTests using MinGW32

```
mkdir build
cd build
cmake .. -G"MinGW Makefiles"
mingw32-make
mingw32-make test
```

Note that this builds 64-bit binaries on 64-bit Windows. To build 32-bit
binaries on 64-bit Windows, replace in above:

```
cmake .. -DCMAKE_C_FLAGS="-m32" -DCMAKE_ASM_NASM_OBJECT_FORMAT="win32" -G"MinGW Makefiles"
```

### Example: building and running go tests using MinGW32

```
mkdir build
cd build
cmake .. -G"MinGW Makefiles"
mingw32-make sointu
cd ..
go test ./...
```

Running `mingw32-make sointu` only builds the static library that go needs. This
is a lot faster than building all the CTests.

If you plan to build the Sointu library for using it from the Go side, you
*must* build in the build/ directory, as bridge.go assumes the library can be
found from build/.

> :warning: At the moment, you must have gcc (e.g. mingw32 on Windows) to build
the project. This is because cgo (the bridge to call c from go) requires gcc
compiler, and the rest of the project uses the go code to automatically build
the .asm from the .yml test cases. A solution that drops the need for CMake and
gcc is in the works; it likely involves dropping precompiled binaries for the
most popular platforms in the repository, which should also allow `go get` the
project. Regardless, it is possible to build the test cases using Visual Studio,
after building the library using mingw32. Luckily, the prebuilt binaries will be
only few tens of KB; this is a 4k synth project after all.

> :warning: **If you are using MinGW and Yasm**: Yasm 1.3.0 (currently still the
latest stable release) and GNU linker do not play nicely along, trashing the BSS
layout. See
[here](https://tortall.lighthouseapp.com/projects/78676/tickets/274-bss-problem-with-windows-win64)
and the fix
[here](https://github.com/yasm/yasm/commit/1910e914792399137dec0b047c59965207245df5).
Use a newer nightly build of yasm that includes the fix. The linker had placed
our synth object overlapping with DLL call addresses; very funny stuff to debug.

New features since fork
-----------------------
  - **Compiler**. Written in go. The input is a .yml file and the output is an
    .asm. It works by inputting the song data to the excellent go
    `text/template` package, effectively working as a preprocessor. This allows
    quite powerful combination: we can handcraft the assembly code to keep the
    entropy as low as possible, yet we can call arbitrary go functions as
    "macros".
  - **Tracker**. Written in go. A prototype exists.
  - **Supports 32 and 64 bit builds**. The 64-bit version is done with minimal
    changes to get it work, using template macros to change the lines between
    32-bit and 64-bit modes. Mostly, it's as easy as writing {{.AX}} instead of
    eax; the macro {{.AX}} compiles to eax in 32-bit and rax in 64-bit.
  - **Supports Windows, Linux and MacOS**. On all three 64-bit platforms, all
    tests are passing. Additionally, all tests are passing on windows 32.
  - **New units**. For example: bit-crusher, gain, inverse gain, clip, modulate
    bpm (proper triplets!), compressor (can be used for side-chaining).
  - **Per instrument polyphonism**. An instrument has the possibility to have
    any number of voices, meaning in practice that multiple voices can reuse the
    same opcodes. So, you can have a single instrument with three voices, and
    three tracks that use this instrument, to make chords. See
    [here](tests/test_chords.asm) for an example and [here](templates/patch.yml)
    for the implementation. The maximum total number of voices will be 32: you
    can have 32 monophonic instruments or any combination of polyphonic
    instruments adding up to 32.
  - **Any number of voices per track**. A single track can trigger more than one
    voice. At every note, a new voice from the assigned voices is triggered and
    the previous released. Combined with the previous, you can have a single
    track trigger 3 voices and all these three voices use the same instrument,
    useful to do polyphonic arpeggios (see [here](tests/test_polyphony.yml)).
    Not only that, a track can even trigger voices of different instruments,
    alternating between these two; maybe useful for example as an easy way to
    alternate between an open and a closed hihat.
  - **Easily extensible**. Instead of %ifdef hell, the primary extension
    mechanism will be through new opcodes for the virtual machine. Only the
    opcodes actually used in a song are compiled into the virtual machine. The
    goal is to try to write the code so that if two similar opcodes are used,
    the common code in both is reused by moving it to a function. Macro and
    linker magic ensure that also helper functions are only compiled in if they
    are actually used.
  - **Songs are YAML files**. These markup files are simple data files,
    describing the tracks, patterns and patch structure (see
    [here](tests/test_oscillat_trisaw.yml) for an example). The sointu-cli
    compiler then reads these files and compiles them into .asm code. This has
    the nice implication that, in future, there will be no need for a binary
    format to save patches, nor should you need to commit .o or .asm to repo:
    just put the .yml in the repo and automate the .yml -> .asm -> .o steps
    using sointu-cli & nasm.
  - **Harmonized support for stereo signals**. Every opcode supports a stereo
    variant: the stereo bit is hidden in the least significant bit of the
    command stream and passed in carry to the opcode. This has several nice
    advantages: 1) the opcodes that don't need any parameters do not need an
    entire byte in the value stream to define whether it is stereo; 2) stereo
    variants of opcodes can be implemented rather efficiently; in some cases,
    the extra cost of stereo variant is only 5 bytes (uncompressed). 3) Since
    stereo opcodes usually follow stereo opcodes (and mono opcodes follow mono
    opcodes), the stereo bits of the command bytes will be highly correlated and
    if crinkler or any other modeling compressor is doing its job, that should
    make them highly predictable i.e. highly compressably.
  - **Test-driven development**. Given that 4klang was already a mature project,
    the first thing actually implemented was a set of regression tests to avoid
    breaking everything beyond any hope of repair. Done, using go test (runs the
    .yml regression tests through the library) and CTest (compiles each .yml
    into executable and ensures that when run like this, the test case produces
    identical output). The tests are also ran in the cloud using github actions.
  - **Arbitrary signal routing**. SEND (used to be called FST in 4klang) opcode
    normally sends the signal as a modulation to another opcode. But with the
    new RECEIVE opcode, you just receive the plain signal there. So you can
    connect signals in an arbitrary way. Actually, 4klang could already do this
    but in a very awkward way: it had FLD (load value) opcode that could be
    modulated; FLD 0 with modulation basically achieved what RECEIVE does,
    except that RECEIVE can also handle stereo signals. Additionally, we have
    OUTAUX, AUX and IN opcodes, which route the signals through global main or
    aux ports, more closer to how 4klang does. But this time we have 8 mono
    ports / 4 stereo ports, so even this method of routing is unlikely to run
    out of ports in small intros.
  - **Pattern length does not have to be a power of 2**.
  - **Sample-based oscillators, with samples imported from gm.dls**. Reading
    gm.dls is obviously Windows only, but the sample mechanism can be used also
    without it, in case you are working on a 64k and have some kilobytes to
    spare. See [this example](tests/test_oscillat_sample.yml), and this Python
    [script](scripts/parse_gmdls.py) parses the gm.dls file and dumps the sample
    offsets from it.
  - **Unison oscillators**. Multiple copies of the oscillator running slightly
    detuned and added up to together. Great for trance leads (supersaw). Unison
    of up to 4, or 8 if you make stereo unison oscillator and add up both left
    and right channels. See [this example](tests/test_oscillat_unison.yml).
  - **Compiling as a library**. The API is very rudimentary, a single function
    render, and between calls, the user is responsible for manipulating the
    synth state in a similar way as the actual player does (e.g. triggering/
    releasing voices etc.)
  - **Calling Sointu as a library from Go language**. The Go API is slighty more
    sane than the low-level library API, offering more Go-like experience.

Future goals
------------

  - **Find a more general solution for skipping opcodes / early outs**. It might
    be a new opcode "skip" that skips from the opcode to the next out in case
    the signal entering skip and the signal leaving out are both close to zero.
    Need to investigate the best way to implement this.
  - **Even more opcodes**. Some potentially useful additions could be:
    - Equalizer / more flexible filters
    - Very slow filters (~ DC-offset removal). Can be implemented using a single
      bit flag in the existing filter
    - Arbitrary envelopes; for easier automation.
  - **MIDI support for the tracker**.
  - **Reintroduce the sync mechanism**. 4klang could export the envelopes of all
    instruments at a 256 times lower frequency, with the purpose of using them
    as sync data. This feature was removed at some point, but should be
    reintroduced at some point. Need to investigate the best way to implement
    this; maybe a "sync" opcode that save the current signal from the stack? Or
    reusing sends/outs and having special sync output ports, allowing easily
    combining multiple signals into one sync. Oh, and we probably should dump
    the whole thing also as a texture to the shader; to fly through the song, in
    a very literal way.

Crazy ideas
-----------
  - **Using Sointu as a sync-tracker**. Similar to [GNU
    Rocket](https://github.com/yupferris/gnurocket), but (ab)using the tracker
    we already have for music. We could define a generic RPC protocol for Sointu
    tracker send current sync values and time; one could then write a debug
    version of a 4k intro that merely loads the shader and listens to the RPC
    messages, and then draws the shader with those as the uniforms. Then, during
    the actual 4k intro, just render song, get sync data from Sointu and send as
    uniforms to shader. A track with two voices, triggering an instrument with a
    single envelope and a slow filter can even be used as a cheap smooth
    interpolation mechanism.
  - **Support WASM targets with the compiler**. It should not be impossible to
    reimplement the x86 core with WAT equivalent. It would be nice to make it
    work (almost) exactly like the x86 version, so one could just track the song
    with Sointu tools and export the song to WAT using sointu-cli.
  - **Hack deeper into audio sources from the OS**. Speech synthesis, I'm eyeing
    at you.

Anti-goals
----------
  - **Ability to run Sointu as a DAW plugin (VSTi, AU, LADSPA and DSSI...)**.
    None of these plugin technologies are cross-platform and they are full of
    proprietary technologies. In particular, since Sointu was initiated after
    Steinberg ceased to give out VSTi2 licenses, there is currently no legal or
    easy way to compile it as a VSTi2 plugin. I downloaded the VSTi3 API and,
    nope, sorry, I don't want to spend my time on it. And Renoise supports only
    VSTi2... There is [JUCE](https://juce.com/), but it is again a mammoth and
    requires apparently pretty deep integration in build system in the form of
    Projucer. If someone comes up with a light-weight way and easily
    maintainable way to make the project into DAW plugin, I may reconsider. For
    now, if you really must, we aim to support MIDI.

Design philosophy
-----------------

  - Make sure the assembly code is readable after compiling: it should have
    liberally comments *in the outputted .asm file*. This allows humans to study
    the outputted code and figure out more easily if there's still way to
    squueze out instructions from the code.
  - Instead of prematurely adding %ifdef toggles to optimize away unused
    features, start with the most advanced featureset and see if you can
    implement it in a generalized way. For example, all the modulations are now
    added into the values when they are converted from integers, in a
    standardized way. This got rid of most of the %ifdefs in 4klang. Also, with
    no %ifdefs cluttering the view, many opportunities to shave away
    instructions became apparent. Also, by making the most advanced synth
    cheaply available to the scene, we promote better music in future 4ks :)
  - Size first, speed second. Speed will only considered if the situation
    becomes untolerable.
  - Benchmark optimizations. Compression results are sometimes slightly
    nonintuitive so alternative implementations should always be benchmarked
    e.g. by compiling and linking a real-world song with
    [Leviathan](https://github.com/armak/Leviathan-2.0) and observing how the
    optimizations affect the byte size.

Background and history
----------------------

[4klang](https://github.com/hzdgopher/4klang) development was started in 2007 by
Dominik Ries (gopher) and Paul Kraus (pOWL) of Alcatraz. The
[write-up](http://zine.bitfellas.org/article.php?zine=14&id=35) will still be
helpful for anyone looking to understand how 4klang and Sointu use the FPU stack
to manipulate the signals. Since then, 4klang has been used in countless of
scene productions and people use it even today.

However, 4klang is not actively developed anymore and the polyphonism was never
implemented in a very well engineered way (you can have exactly 2 voices per
instrument if you enable it). Also, reading through the code, I spotted several
avenues to squeeze away more bytes. These observations triggered project Sointu.
That, and I just wanted to learn x86 assembly, and needed a real-world project
to work on.

What's with the name
--------------------

"Sointu" means a chord, in Finnish; a reference to the polyphonic capabilities
of the synth. Also, I assume we have all learned by now what "klang" means in
German, so I thought it would fun to learn some Finnish for a change. And
[there's](https://www.pouet.net/prod.php?which=53398)
[enough](https://www.pouet.net/prod.php?which=75814)
[klangs](https://www.pouet.net/prod.php?which=85351) already.

Credits
-------

The original 4klang was developed by Dominik Ries
([gopher](https://github.com/hzdgopher/4klang)) and Paul Kraus (pOWL) of
Alcatraz. :heart:

Sointu was initiated by Veikko Sariola (pestis/bC!).

Apollo/bC! put the project on the path to Go, and wrote the prototype of the
tracker GUI.

PoroCYon's [4klang fork](https://github.com/PoroCYon/4klang) inspired the macros
to better support cross-platform asm.
