# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]
### Added
- An instrument (set of opcodes & accompanying values) can have any number of voices.
- A track can trigger any number of voices (polyphonism).
- Pattern length does not have to be a power of 2.
- Macros for defining patches, so that only the necessary parts of the synth are compiled in.
- Harmonized support for stereo signals: every opcode supports stereo variant.
- New opcodes: bit-crusher, gain, inverse gain, clip, speed (bpm modulation), compressor.
- Support for sample-based oscillators; samples loaded from gm.dls.
- Unison oscillators: multiple copies of the oscillator running sligthly detuned and added up to together.
- Support for 32 and 64 bit builds.
- Regression tests for opcodes, using CTests.
- Switch to CMake for builds.

[Unreleased]: https://github.com/vsariola/sointu/compare/4klang-3.11...HEAD