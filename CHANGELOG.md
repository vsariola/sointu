# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## Unreleased
### Added
- Saving and loading instruments
- Non-platform native file save and load dialogs, for more reliable support across platforms
- Comment field to instruments
- The repository has now a few example instruments
- Ability to reorder tracks

## v0.1.0
### Added
- An instrument (set of opcodes & accompanying values) can have any number of voices.
- A track can trigger any number of voices, releasing the previous when new one is triggered.
- Pattern length does not have to be a power of 2.
- Only the necessary opcodes and functions of the synth are compiled in the final executable.
- Harmonized support for stereo signals: every opcode supports stereo variant.
- New opcodes: crush, gain, inverse gain, clip, speed (bpm modulation), compressor.
- Support for sample-based oscillators (samples loaded from gm.dls).
- Unison oscillators: multiple copies of the oscillator running with different detuning and added up to together.
- Support for 32 and 64 bit builds.
- Support different platforms: Windows, Linux and Mac (Intel).
- Experimental support for compiling songs into WebAssembly.
- Switch to CMake for builds.
- Regression tests for every VM instruction, using CTests.
- Compiling as a static library & an API to call Sointu
- Running all tests (win/linux/mac/wasm) in the cloud, using Github workflows
- Tools written in Go-lang:
  - a tracker for composing songs as .yml
  - a command line utility to convert .yml songs to .asm
  - a command line utility to play the songs on command line

[Unreleased]: https://github.com/vsariola/sointu/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/vsariola/sointu/compare/4klang-3.11...v0.1.0