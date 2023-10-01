# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## Unreleased
### Added
- Save the GUI state periodically to a recovery file and load it on
  startup of the app, if present. The recovery file is located in the
  home directory of the user.

### Fixed
- The sointu-vsti-native plugin has different plugin ID and plugin name
  to not confuse it with the non-native one
- The VSTI waits for the gioui actually have quit when closing the
  plugin

## v0.2.0
### Added
- Saving and loading instruments
- Comment field to instruments
- Ability to reorder tracks
- Add menu command to delete all unused data from song file
- Ability to search a unit by typing its name
- Ability to run sointu as a vsti plugin, inside vsti host
- Ability to lock delay relative to beat duration
- Ability to import 4klang patches (.4kp) and instruments (.4ki)
- The repository has example instruments, including all patches and
  instruments from 4klang
- The compiler templates are embedded in the sointu-compile, so no
  installation is needed beyond copying sointu-compile to PATH
- Ability to select multiple units and cut, copy & paste them
- Mousewheel adjusts unit parameters
- Tooltips to many buttons
- Support for gm.dls samples in the go-written virtual machine
- x86 and C written examples how to play a sointu song on various
  platforms. On Windows, the examples can optionally be linked with
  Crinkler to get Crinkler reports.

### Fixed
- Unnamed instruments with multiple voices caused crashes
- In the native version, exceeding the 64 delaylines caused crashes
- wat2wasm nowadays uses funcref instead of anyfunc
- In the WebAssembly core, $WRK was messed after stereo oscillators,
  making modulations not work
- The Webassembly implementation of mono version of the "out" unit

### Changed
- The release flag in the voice is now a sustain flag i.e. the logic has
  been inverted. This was done so that when the synth is initialized
  with zeros, all voices start with sustain = 0 i.e. in released state.
- The crush resolution is now in bits instead of linear range; this is a
  breaking change and changes the meaning of the resolution values. But
  now there are more usable values in the resolution.

## v0.1.0
### Added
- An instrument (set of opcodes & accompanying values) can have any
  number of voices.
- A track can trigger any number of voices, releasing the previous when
  new one is triggered.
- Pattern length does not have to be a power of 2.
- Only the necessary opcodes and functions of the synth are compiled in the final executable.
- Harmonized support for stereo signals: every opcode supports stereo
  variant.
- New opcodes: crush, gain, inverse gain, clip, speed (bpm modulation),
  compressor.
- Support for sample-based oscillators (samples loaded from gm.dls).
- Unison oscillators: multiple copies of the oscillator running with
  different detuning and added up to together.
- Support for 32 and 64 bit builds.
- Support different platforms: Windows, Linux and Mac (Intel).
- Experimental support for compiling songs into WebAssembly.
- Switch to CMake for builds.
- Regression tests for every VM instruction, using CTests.
- Compiling as a static library & an API to call Sointu
- Running all tests (win/linux/mac/wasm) in the cloud, using Github
  workflows
- Tools written in Go-lang:
  - a tracker for composing songs as .yml
  - a command line utility to convert .yml songs to .asm
  - a command line utility to play the songs on command line

[Unreleased]: https://github.com/vsariola/sointu/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/vsariola/sointu/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/vsariola/sointu/compare/4klang-3.11...v0.1.0