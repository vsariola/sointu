# Sointu
![Tests](https://github.com/vsariola/sointu/workflows/Tests/badge.svg)
![Binaries](https://github.com/vsariola/sointu/workflows/Binaries/badge.svg)

A cross-architecture and cross-platform modular software synthesizer for small
intros, forked from [4klang](https://github.com/hzdgopher/4klang). Targetable
architectures include 386, amd64, and WebAssembly; targetable platforms include
Windows, Mac, Linux (and related) + browser.

- [User manual](https://github.com/vsariola/sointu/wiki) is in the Wiki
- [Discussions](https://github.com/vsariola/sointu/discussions) is for asking
  help, sharing patches/instruments and brainstorming ideas
- [Issues](https://github.com/vsariola/sointu/issues) is for reporting bugs

Installation
------------

You can either:

  1) Download the latest build from the master branch from the
     [actions](https://github.com/vsariola/sointu/actions) (find workflow
     "Binaries" and scroll down for .zip files containing the artifacts.
     **Note:** You have to be logged into Github to download artifacts!

 or

  2) Download the prebuilt release binaries from the
     [releases](https://github.com/vsariola/sointu/releases). Then just run one
     of the executables or, in the case of the VST plugins library files, copy
     them wherever you keep you VST2 plugins.

The pre 1.0 version tags are mostly for reference: no backwards
compatibility will be guaranteed while upgrading to a newer version.
Backwards compatibility will be attempted from 1.0 onwards.

**Uninstallation**: Sointu stores recovery data in OS-specific folders
e.g. `AppData/Roaming/Sointu` on Windows. For clean uninstall, delete
also this folder. See [here](https://pkg.go.dev/os#UserConfigDir) where
to find those folders on other platforms.

Summary
-------

Sointu is work-in-progress. It is a fork and an evolution of
[4klang](https://github.com/hzdgopher/4klang), a modular software synthesizer
intended to easily produce music for 4k intros &mdash; small executables with a
maximum filesize of 4096 bytes containing realtime audio and visuals. Like
4klang, the sound is produced by a virtual machine that executes small bytecode
to produce the audio; however, by now the internal virtual machine has been
heavily rewritten and extended. It is actually extended so much that you will
never fit all the features at the same time in a 4k intro, but a fairly capable
synthesis engine can already be fitted in 600 bytes (386, compressed), with
another few hundred bytes for the patch and pattern data.

Sointu consists of two core elements:
- A cross-platform synth-tracker that runs as either VSTi or stand-alone
  app for composing music, written in [go](https://golang.org/). The app
  is still heavily work in progress. The app exports the projects as
  .yml files.
- A compiler, likewise written in go, which can be invoked from the command line
  to compile these .yml files into .asm or .wat code. For x86/amd64, the
  resulting .asm can be then compiled by [nasm](https://www.nasm.us/). For
  browsers, the resulting .wat can be compiled by
  [wat2wasm](https://github.com/WebAssembly/wabt).

This is how the current prototype app looks like:

![Screenshot of the tracker](screenshot.png)

Building
--------

Various aspects of the project have different tool dependencies, which are
listed below.

### Sointu-track

This is the stand-alone version of the synth-tracker. Sointu-track uses
the [gioui](https://gioui.org/) for the GUI and [oto](https://github.com/hajimehoshi/oto)
for the audio, so the portability is currently limited by these.

#### Prerequisites

- [go](https://golang.org/)
- If you want to also use the x86 assembly written synthesizer, to test that the
  patch also works once compiled:
   - Follow the instructions to build the [x86 native virtual machine](#native-virtual-machine)
     before building the tracker.
   - cgo compatible compiler e.g. [gcc](https://gcc.gnu.org/). On
     windows, you best bet is [MinGW](http://www.mingw.org/). We use the [tdm-gcc](https://jmeubank.github.io/tdm-gcc/).
     The compiler can be in PATH or you can use the environment variable
     `CC` to help go find the compiler.
   - Setting environment variable `CGO_ENABLED=1` is a good idea,
     because if it is not set and go fails to find the compiler, go just
     excludes all files with `import "C"` from the build, resulting in
     lots of errors about missing types.

#### Running

```
go run cmd/sointu-track/main.go
```

#### Building an executable

```
go build -o sointu-track.exe cmd/sointu-track/main.go
```

On other platforms than Windows, replace `-o sointu-track.exe` with
`-o sointu-track`.

If you want to include the [x86 native virtual machine](#native-virtual-machine),
add `-tags=native` to all the commands e.g.

```
go build -o sointu-track.exe -tags=native cmd/sointu-track/main.go
```

### Sointu-vsti

This is the VST instrument plugin version of the tracker, compiled into
a dynamically linked library and ran inside a VST host.

#### Prerequisites

- [go](https://golang.org/)
- cgo compatible compiler e.g. [gcc](https://gcc.gnu.org/). On windows,
  you best bet is [MinGW](http://www.mingw.org/). We use the [tdm-gcc](https://jmeubank.github.io/tdm-gcc/).
  The compiler can be in PATH or you can use the environment variable
  `CC` to help go find the compiler.
- Setting environment variable `CGO_ENABLED=1` is a good idea, because
  if it is not set and go fails to find the compiler, go just excludes
  all files with `import "C"` from the build, resulting in lots of
  errors about missing types.
- If you want to build the VSTI with the native x86 assembly written synthesizer:
   - Follow the instructions to build the [x86 native virtual machine](#native-virtual-machine)
     before building the plugin itself

#### Building

```
go build -buildmode=c-shared -tags=plugin -o sointu-vsti.dll .\cmd\sointu-vsti\
```

On other platforms than Windows, replace `-o sointu-vsti.dll` appropriately e.g.
`-o sointu-vsti.so`; so far, the VST instrument has been built & tested on
Windows and Linux.

Notice the `-tags=plugin` build tag definition. This is required by the [vst2
library](https://github.com/pipelined/vst2); otherwise, you will get a lot of
build errors.

Add `-tags=native,plugin` to use the [x86 native virtual
machine](#native-virtual-machine) instead of the virtual machine written in Go.

### Sointu-compile

The command line interface to it is [sointu-compile](cmd/sointu-compile/main.go)
and the actual code resides in the [compiler](vm/compiler/) package, which is an
ordinary [go](https://golang.org/) package with no other tool dependencies.

#### Running

```
go run cmd/sointu-compile/main.go
```

#### Building an executable

```
go build -o sointu-compile.exe cmd/sointu-compile/main.go
```

On other platforms than Windows, replace `-o sointu-compile.exe` with
`-o sointu-compile`.

#### Usage

The compiler can then be used to compile a .yml song into .asm and .h files. For
example:

```
sointu-compile -o . -arch=386 tests/test_chords.yml
nasm -f win32 test_chords.asm
```

WebAssembly example:

```
sointu-compile -o . -arch=wasm tests/test_chords.yml
wat2wasm test_chords.wat
```

If you are looking for an easy way to compile an executable from a Sointu song
(e.g. for a executable music compo), take a look at [NR4's Python-based
tool](https://github.com/LeStahL/sointu-executable-msx) for it.

#### Examples

The folder `examples/code` contains usage examples for Sointu with winmm and
dsound playback under Windows and asound playback under Unix. Source code is
available in C and x86 assembly (win32, elf32 and elf64 versions).

To build the examples, use `ninja examples`.

If you want to target smaller executable sizes, using a compressing linker like
[Crinkler](https://github.com/runestubbe/Crinkler) on Windows is recommended.

The linux examples use ALSA and need libasound2-dev (or libasound2-dev:386)
installed. The 386 version also needs pipewire-alsa:386 installed, which is not
there by default.

### Native virtual machine

The native bridge allows Go to call the Sointu compiled x86 native virtual
machine, through cgo, instead of using the Go written bytecode interpreter. With
the latest Go compiler, the native virtual machine is actually slower than the
Go-written one, but importantly, the native virtual machine allows you to test
that the patch also works within the stack limits of the x87 virtual machine,
which is the VM used in the compiled intros. In the tracker/VSTi, you can switch
between the native synth and the Go synth under the CPU panel in the Song
settings.

Before you can actually run it, you need to build the bridge using CMake (thus,
***this will not work with go get***).

#### Prerequisites

- [CMake](https://cmake.org)
- [nasm](https://www.nasm.us/)
- *cgo compatible compiler* e.g. [gcc](https://gcc.gnu.org/). On windows, you
  best bet is [MinGW](http://www.mingw.org/). We use the
  [tdm-gcc](https://jmeubank.github.io/tdm-gcc/)

The last point is because the command line player and the tracker use
[cgo](https://golang.org/cmd/cgo/) to interface with the synth core, which is
compiled into a library. The cgo bridge resides in the package
[bridge](vm/compiler/bridge/).

#### Building

Assuming you are using [ninja](https://ninja-build.org/):

```
mkdir build
cd build
cmake .. -GNinja
ninja sointu
```

> :warning: *you must build the library inside a directory called 'build' at the
> root of the project*. This is because the path where cgo looks for the library
> is hard coded to point to build/ in the go files.

Running `ninja sointu` only builds the static library that Go needs. This is a
lot faster than building all the CTests.

You and now run all the Go tests, even the ones that test the native bridge.
From the project root folder, run:

```
go test ./...
```

Play a song from the command line:
```
go run -tags=native cmd/sointu-play/main.go tests/test_chords.yml
```

> :warning: Unlike the x86/amd64 VM compiled by Sointu, the Go written VM
> bytecode interpreter uses a software stack. Thus, unlike x87 FPU stack, it is
> not limited to 8 items. If you intent to compile the patch to x86/amd64
> targets, make sure not to use too much stack. Keeping at most 5 signals in the
> stack is presumably fine (reserving 3 for the temporary variables of the
> opcodes). In future, the app should give warnings if the user is about to
> exceed the capabilities of a target platform.

> :warning: **If you are using Yasm instead of Nasm, and you are using MinGW**:
> Yasm 1.3.0 (currently still the latest stable release) and GNU linker do not
> play nicely along, trashing the BSS layout. The linker had placed our synth
> object overlapping with DLL call addresses; very funny stuff to debug. See
> [here](https://tortall.lighthouseapp.com/projects/78676/tickets/274-bss-problem-with-windows-win64)
> and the fix
> [here](https://github.com/yasm/yasm/commit/1910e914792399137dec0b047c59965207245df5).
> Since Nasm is nowadays under BSD license, there is absolutely no reason to use
> Yasm. However, if you do, use a newer nightly build of Yasm that includes the
> fix.

### Tests

There are [regression tests](tests/) that are built as executables,
testing that they work the same way when you would link them in an
intro.

#### Prerequisites

- [go](https://golang.org/)
- [CMake](https://cmake.org) with CTest
- [nasm](https://www.nasm.us/)
- Your favorite CMake compatible c-compiler & build tool. Results have been
  obtained using Visual Studio 2019, gcc&make on linux, MinGW&mingw32-make, and
  ninja&AppleClang.

#### Building and running

Assuming you are using [ninja](https://ninja-build.org/):

```
mkdir build
cd build
cmake .. -GNinja
ninja
ninja test
```

Note that this builds 64-bit binaries on 64-bit Windows. To build 32-bit
binaries on 64-bit Windows, replace in above:

```
cmake .. -DCMAKE_C_FLAGS="-m32" -DCMAKE_ASM_NASM_OBJECT_FORMAT="win32" -GNinja
```

Another example: on Visual Studio 2019 Community, just open the folder, choose
either Debug or Release and either x86 or x64 build, and hit build all.

### WebAssembly tests

These are automatically invoked by CTest if [node](https://nodejs.org) and
[wat2wasm](https://github.com/WebAssembly/wabt) are found in the path.

New features since fork
-----------------------

  - **New units**. For example: bit-crusher, gain, inverse gain, clip, modulate
    bpm (proper triplets!), compressor (can be used for side-chaining).
  - **Compiler**. Written in go. The input is a .yml file and the output is an
    .asm. It works by inputting the song data to the excellent go
    `text/template` package, effectively working as a preprocessor. This allows
    quite powerful combination: we can handcraft the assembly code to keep the
    entropy as low as possible, yet we can call arbitrary go functions as
    "macros". The templates are [here](vm/compiler/templates/) and the compiler lives
    [here](vm/compiler/).
  - **Tracker**. Written in go. Can run either as a stand-alone app or a vsti
    plugin.
  - **Supports 32 and 64 bit builds**. The 64-bit version is done with minimal
    changes to get it work, using template macros to change the lines between
    32-bit and 64-bit modes. Mostly, it's as easy as writing {{.AX}} instead of
    eax; the macro {{.AX}} compiles to eax in 32-bit and rax in 64-bit.
  - **Supports compiling into WebAssembly**. This is a complete reimplementation
    of the core, written in WebAssembly text format (.wat).
  - **Supports Windows, Linux and MacOS**. On all three 64-bit platforms, all
    tests are passing. Additionally, all tests are passing on windows 32.
  - **Per instrument polyphonism**. An instrument has the possibility to have
    any number of voices, meaning that multiple voices can reuse the same
    opcodes. So, you can have a single instrument with three voices, and three
    tracks that use this instrument, to make chords. See
    [here](tests/test_chords.yml) for an example and
    [here](vm/compiler/templates/amd64-386/patch.asm) for the implementation.
    The maximum total number of voices is 32: you can have 32 monophonic
    instruments or any combination of polyphonic instruments adding up to 32.
  - **Any number of voices per track**. A single track can trigger more than one
    voice. At every note, a new voice from the assigned voices is triggered and
    the previous released. Combined with the previous, you can have a single
    track trigger 3 voices and all these three voices use the same instrument,
    useful to do polyphonic arpeggios (see [here](tests/test_polyphony.yml)).
    Not only that, a track can even trigger voices of different instruments,
    alternating between these two; maybe useful for example as an easy way to
    alternate between an open and a closed hihat.
  - **Reasonably easily extensible**. Instead of %ifdef hell, the primary
    extension mechanism is through new opcodes for the virtual machine. Only the
    opcodes actually used in a song are compiled into the virtual machine. The
    goal is to try to write the code so that if two similar opcodes are used,
    the common code in both is reused by moving it to a function. Macro and
    linker magic ensure that also helper functions are only compiled in if they
    are actually used.
  - **Songs are YAML files**. These markup files are simple data files,
    describing the tracks, patterns and patch structure (see
    [here](tests/test_oscillat_trisaw.yml) for an example). The sointu-compile
    then reads these files and compiles them into .asm code. This has the nice
    implication that, in future, there will be no need for a binary format to
    save patches, nor should you need to commit .o or .asm to repo: just put the
    .yml in the repo and automate the .yml -> .asm -> .o steps using
    sointu-compile & nasm.
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
    make them highly predictable i.e. highly compressable.
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
  - **Sample-based oscillators, with samples imported from gm.dls**. The
    gm.dls is available from system folder only on Windows, but the
    non-native tracker looks for it also in the current folder, so
    should you somehow magically get hold of gm.dls on Linux or Mac, you
    can drop it in the same folder with the tracker. See [this example](tests/test_oscillat_sample.yml),
    and this go generate [program](cmd/sointu-generate/main.go) parses
    the gm.dls file and dumps the sample offsets from it.
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
  - **A bytecode interpreter written in pure go**. With the latest Go compiler,
    it's slightly faster hand-written one using x87 opcodes. With this, the
    tracker is ultraportable and does not need cgo calls.

Design philosophy
-----------------

  - Make sure the assembly code is readable after compiling: it should have
    liberally comments *in the outputted .asm file*. This allows humans to study
    the outputted code and figure out more easily if there's still way to
    squeeze out instructions from the code.
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
    [one of the examples](examples/code/C) and observing how the optimizations
    affect the byte size.

Background and history
----------------------

[4klang](https://github.com/hzdgopher/4klang) development was started in 2007 by
Dominik Ries (gopher) and Paul Kraus (pOWL) of Alcatraz. The
[write-up](http://zine.bitfellas.org/article.php?zine=14&id=35) will still be
helpful for anyone looking to understand how 4klang and Sointu use the FPU stack
to manipulate the signals. Since then, 4klang has been used in countless of
scene productions and people use it even today.

However, 4klang seems not to be actively developed anymore and polyphonism was
implemented only in a rather limited way (you could have exactly 2 voices per
instrument if you enable it). Also, reading through the code, I spotted several
avenues to squeeze away more bytes. These observations triggered project Sointu.
That, and I just wanted to learn x86 assembly, and needed a real-world project
to work on.

What's with the name
--------------------

"Sointu" means a chord, in Finnish; a reference to the polyphonic capabilities
of the synth. I assume we have all learned by now what "klang" means in German,
so I thought it would fun to learn some Finnish for a change. And
[there's](https://www.pouet.net/prod.php?which=53398)
[enough](https://www.pouet.net/prod.php?which=75814)
[klangs](https://www.pouet.net/prod.php?which=85351) already.

Prods using Sointu
------------------

  - [Adam](https://github.com/vsariola/adam) by brainlez Coders! My first
    test-driving of Sointu. The repository has some ideas how to integrate
    Sointu to the build chain.
  - [Roadtrip](https://www.pouet.net/prod.php?which=94105) by LJ & Virgill
  - [|](https://www.pouet.net/prod.php?which=94721) by epoqe. Likely the first
    Linux 4k intro using sointu.
  - [Physics Girl St.](https://www.pouet.net/prod.php?which=94890) by Team210
  - [Delusions of mediocrity](https://www.pouet.net/prod.php?which=95222) by
    mrange & Virgill
  - [Xorverse](https://www.pouet.net/prod.php?which=95221) by Alcatraz
  - [l'enveloppe](https://www.pouet.net/prod.php?which=95215) by Team210 & epoqe
  - [Phosphorescent Purple Pixel Peaks](https://www.pouet.net/prod.php?which=96198) by mrange & Virgill
  - [21](https://demozoo.org/music/338597/) by NR4 / Team210
  - [Tausendeins](https://www.pouet.net/prod.php?which=96192) by epoqe & Team210
  - [Radiant](https://www.pouet.net/prod.php?which=97200) by Team210
  - [Aurora Florae](https://www.pouet.net/prod.php?which=97516) by Team210 and
    epoqe
  - [Night Ride](https://www.pouet.net/prod.php?which=98212) by Ctrl-Alt-Test &
    Alcatraz
  - [Bicolor Challenge](https://demozoo.org/competitions/19410/) with [Sointu
    song](https://files.scene.org/view/parties/2024/deadline24/bicolor_challenge/wayfinder_-_bicolor_soundtrack.zip)
    provided by wayfinder
  - [napolnitel](https://www.pouet.net/prod.php?which=104336) by jetlag

Contributing
------------

Pull requests / suggestions / issues welcome, through Github! Or just DM
me on Discord (see contact information below).

License
-------

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.

Contact
-------

Veikko Sariola - pestis_bc on Demoscene discord - firstname.lastname@gmail.com

Project Link: [https://github.com/vsariola/sointu](https://github.com/vsariola/sointu)

Credits
-------

The original 4klang: Dominik Ries ([gopher/Alcatraz](https://github.com/hzdgopher/4klang))
& Paul Kraus (pOWL/Alcatraz) :heart:

Sointu: Veikko Sariola (pestis/bC!), [Apollo/bC!](https://github.com/moitias),
[NR4/Team210](https://github.com/LeStahL/),
[PoroCYon](https://github.com/PoroCYon/4klang),
[kendfss](https://github.com/kendfss), [anticore](https://github.com/anticore),
[qm210](https://github.com/qm210), [reaby](https://github.com/reaby)
