# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]
### Added
- The loudness detection is now LUFS and peak detection is based on oversampled
  true peak detection
- Oscilloscope to visualize the outputted waveform ([#61][i61])
- Toggle button to keep instruments and tracks linked, and buttons to to split
  instruments and tracks with more than 1 voice into parallel ones
  ([#163][i163], [#157][i157])
- Mute and solo toggles for instruments ([#168][i168])
- Compressor displays threshold and invgain in dB
- Dragging mouse to select rectangles in the tables
- The standalone tracker can open a MIDI port for receiving MIDI notes
  ([#166][i166])
- The note editor has a button to allow entering notes by MIDI. Polyphony is
  supported if there are tracks available. ([#170][i170])
- Units can have comments, to make it easier to distinguish between units of
  same type within an instrument. These comments are also shown when choosing
  the send target. ([#114][i114])
- A toggle button for copying non-unique patterns before editing. When enabled
  and if the pattern is used in multiple places, the pattern is copied first.
  ([#77][i77])
- User can define own keybindings in `os.UserConfigDir()/sointu/keybindings.yml`
  ([#94][i94], [#151][i151])
- A small number above the instrument name identifies the MIDI channel /
  instrument number, with numbering starting from 1 ([#154][i154])
- The filter unit frequency parameter is displayed in Hz, corresponding roughly
  to the resonant frequency of the filter ([#158][i158])
- Include version info in the binaries, as given be `git describe`. This version
  info is shown as a label in the tracker and can be checked with `-v` flag in
  the command line tools.
- If a parameter is controlled by a `send`, the slider is now colored differently
  and there's a tooltip over the value to see where it comes from and its amount

### Fixed
- We try to honor the MIDI event time stamps, so that the timing between MIDI
  events (as reported to us by RTMIDI) will be correct.
- When unmarshaling the recovery file, the unit parameter maps were "merged"
  with the existing parameter maps, instead of overwriting. This created units
  with unnecessary parameters, which was harmless, but would cause a warning to
  the user.
- When changing a nibble of a hexadecimal note, the note played was the note
  before changing the nibble
- Clicking on low nibble or high nibble of a hex track selects that nibble
  ([#160][i160])
- If units have useless parameters in their parameter maps, from bugs or from a
  malformed yaml file, they are removed and user is warned about it
- Pressing a or 1 when editing note values in hex mode created a note off line
  ([#162][i162])
- Warn about plugin sample rate being different from 44100 only after
  ProcessFloatFunc has been called, so that host has time to set the sample rate
  after initialization.
- Crashes with sample-based oscillators in the 32-bit library, as the pointer to
  sample-table (edi) got accidentally overwritten by detune
- Sample-based oscillators could hard crash if a x87 stack overflow happened
  when calculating the current position in the sample ([#149][i149])
- Numeric updown widget calculated dp-to-px conversion incorrectly, resulting in
  wrong scaling ([#150][i150])
- Empty patch should not crash the native synth ([#148][i148])
- sointu-play does not default to the native synth yet, choose via `-tags=native`
- Most buttons never gain focus, so that clicking a button does not stop
  whatever the user was currently doing and so that the user does not
  accidentally trigger the buttons by having them focused and e.g. hitting space
  ([#156][i156])

### Changed
- The keyboard shortcuts are now again closer to what they were old trackers
  ([#151][i151])
- The stand-alone apps now output floating point sound, as made possible by
  upgrading oto-library to latest version. This way the tracker sound output
  matches the compiled output better, as usually compiled intros output sound in
  floating point. This might be important if OS sound drivers apply some audio
  enhancemenets e.g. compressors to the audio.

## [0.4.1]
### Added
- Clicking the parameter slider also selects that parameter ([#112][i112])
- The vertical and horizontal split bars indicate with a cursor that they can be
  resized ([#145][i145])

### Fixed
- When adding a unit on the last row of the unit list, the editor for entering
  the type of the unit by text did gain focus.
- When inputting a note to the note editor, advance the cursor by step
  ([#144][i144])
- When loading an instrument, make sure the total number of voices does not go
  over the maximum number allowed by vm, and make sure a loaded instrument has
  at least 1 voice
- Potential ID collisions when clearing unit or pasteing instruments
- Assign new IDs to loaded instruments, and fix ID collisions in case they
  somehow still appear ([#146][i146])
- In x86 templates, do not optimize away phase modulations when unisons are used
  even if all phase inputs are zeros, as unisons use the phase modulation
  mechanism to offset the different oscillators
- Do not include delay times in the delay time table if the delay unit is
  disabled ([#139][i139])
- Moved the error and warning popups slightly up so they don't block the unit
  control buttons ([#142][i142])

### Changed
- Do not automatically wrap around the song when playing as it was usually
  unwanted behaviour. There is already the looping mechanism if the user really
  wants to loop the song forever.

## [0.4.0]
### Added
- User can drop preset instruments into `os.UserConfigDir()/sointu/presets/` and
  they appear in the list of presets next time sointu is started.
  ([#125][i125])
- Ability to loop certain section of the song when playing. The loop can be set
  by using the toggle button in the song panel, or by hitting Ctrl+L.
  ([#128][i128])
- Disable units temporarily. The disabled units are shown in gray and are not
  compiled into the patch and are considered for all purposes non-existent.
  Hitting Ctrl-D disables/re-enables the selected unit(s). The yaml file has
  field `disabled: true` for the unit. ([#116][i116])
- Passing a file name on command line immediately tries loading that file ([#122][i122])
- Massive rewrite of the GUI, in particular allowing better copying, pasting and
  scrolling of table-based data (order list and note data).
- Dbgain unit, which allows defining the gain in decibels (-40 dB to +40dB)
- `+` and `-` keys add/subtract values in order editor and pattern editor
  ([#65][i65])
- The function `su_power` is exported so people can reuse it in the main code;
  however, as it assumes the parameter passed in st0 on the x87 stack and
  similarly returns it value in st0 on the x87 stack, to my knowledge there is
  no calling convention that would correspond this behaviour, so you need to
  define a header for it yourself and take care of putting the float value on
  x87 stack.

### Fixed
- Loading a preset did not update the IDs of the newly loaded instrument,
  causing ID collisions and sends target wrong units.
- The x87 native filter unit was denormalizing and eating up a lot of CPU ([#68][i68])
- Modulating delaytime in wasm could crash, because delay time was converted to
  int with i32.trunc_f32_u. Using i32.trunc_f32_s fixed this.
- When recording notes from VSTI, no track was created for instruments that had
  no notes triggered, resulting in misalignment of the tracks from instruments.
- 32-bit su_load_gmdls clobbered ebx, even though __stdcall demands it to be not
  touched ([#130][i130])
- Spaces are allowed in instrument names ([#120][i120])
- Fixed the dropdown for targeting sends making it impossible to choose certain
  ops. This was done just by reducing the default height of popup menus so they
  fit on screen ([#121][i121])
- Warn user about sample rate being other than 44100 Hz, as this lead to weird
  behaviour. Sointu assumes the samplerate always to be 44100 Hz. ([#129][i129])

### Changed
- The scroll wheel behavior for unit integer parameters was flipped: scrolling
  up now increases the value, while scrolling down decreases the value. It was
  vice versa. ([#112][i112])

## [0.3.0]
### Added
- Scroll bars to menus, shown when a menu is too long to fit.
- Save the GUI state periodically to a recovery file and load it on
  startup of the app, if present. The recovery files are located in the
  app config directory (e.g. AppData/Roaming/Sointu on Windows).
- Save the VSTI GUI state to the DAW project file, through GetChunk /
  SetChunk mechanisms.
- Instrument presets. The presets are embedded in the executable and
  there's a button to open a menu to load one of the presets.
- Frequency modulation target for oscillator, as it was in 4klang
- Reverb preset settings for a delay unit, with stereo, left and right
  options

### Fixed
- Crash when running more than one sointu VSTI plugins in parallel
- The scroll bars move in sync with the cursor.
- The stereo version of delay in the go virtual machine (executables / plugins
  not ending with -native) applied the left delay taps on the right channel, and
  the right delay taps on the left channel.
- The sointu-vsti-native plugin has different plugin ID and plugin name
  to not confuse it with the non-native one
- The VSTI waits for the gioui actually have quit when closing the
  plugin

### Changed
- BREAKING CHANGE: The meaning of default modulation mode ("auto") has
  been changed for cross-instrument modulations: it now means "all"
  voices, instead of first voice (which was redundant, as it was same as
  defining voice = 0). This means that for cross-instrument modulations,
  one "all vocies" send gets actually compiled into multiple sends, one
  for each targeted voice. For intra-instrument modulations, the meaning
  stays the same, but the label was changed to "self", to highlight that
  this means the voice modulates only itself and not other voices.

## [0.2.0]
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

## [0.1.0]
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

[Unreleased]: https://github.com/vsariola/sointu/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/vsariola/sointu/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/vsariola/sointu/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/vsariola/sointu/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/vsariola/sointu/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/vsariola/sointu/compare/4klang-3.11...v0.1.0
[i61]: https://github.com/vsariola/sointu/issues/61
[i65]: https://github.com/vsariola/sointu/issues/65
[i68]: https://github.com/vsariola/sointu/issues/68
[i77]: https://github.com/vsariola/sointu/issues/77
[i94]: https://github.com/vsariola/sointu/issues/94
[i112]: https://github.com/vsariola/sointu/issues/112
[i114]: https://github.com/vsariola/sointu/issues/114
[i116]: https://github.com/vsariola/sointu/issues/116
[i120]: https://github.com/vsariola/sointu/issues/120
[i121]: https://github.com/vsariola/sointu/issues/121
[i122]: https://github.com/vsariola/sointu/issues/122
[i125]: https://github.com/vsariola/sointu/issues/125
[i128]: https://github.com/vsariola/sointu/issues/128
[i129]: https://github.com/vsariola/sointu/issues/129
[i130]: https://github.com/vsariola/sointu/issues/130
[i139]: https://github.com/vsariola/sointu/issues/139
[i142]: https://github.com/vsariola/sointu/issues/142
[i144]: https://github.com/vsariola/sointu/issues/144
[i145]: https://github.com/vsariola/sointu/issues/145
[i146]: https://github.com/vsariola/sointu/issues/146
[i148]: https://github.com/vsariola/sointu/issues/148
[i149]: https://github.com/vsariola/sointu/issues/149
[i150]: https://github.com/vsariola/sointu/issues/150
[i151]: https://github.com/vsariola/sointu/issues/151
[i154]: https://github.com/vsariola/sointu/issues/154
[i156]: https://github.com/vsariola/sointu/issues/156
[i157]: https://github.com/vsariola/sointu/issues/157
[i158]: https://github.com/vsariola/sointu/issues/158
[i160]: https://github.com/vsariola/sointu/issues/160
[i162]: https://github.com/vsariola/sointu/issues/162
[i163]: https://github.com/vsariola/sointu/issues/163
[i166]: https://github.com/vsariola/sointu/issues/166
[i168]: https://github.com/vsariola/sointu/issues/168
[i170]: https://github.com/vsariola/sointu/issues/170
