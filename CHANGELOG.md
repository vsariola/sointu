# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]
### Added
- Spectrum analyzer showing the spectrum. When the user has a filter or belleq
  unit selected, it's frequency response is plotted on top. ([#67][i67])
- belleq unit: a bell-shaped second-order filter for equalization. Belleq unit
  takes the center frequency, bandwidth (inverse of Q-factor) and gain (+-40
  dB). Useful for boosting or reducing specific frequency ranges. Kudos to Reaby
  for the initial implementation!
- Multithreaded synths: the user can split the patch up to four threads.
  Selecting the thread can be done on the instrument properties pane.
  Multithreading works only on the multithreaded synths, selectable from the CPU
  panel. Currently the multithreaded rendering has not yet been implemented in
  the compiled player and the thread information is disregarded while compiling
  the song. ([#199][i199])
- Preset explorer, whichs allows 1) searching the presets by name; 2) filtering
  them by category (directory); 3) filtering them by being builtin vs. user;
  4) filtering them if they need gm.dls (for Linux/Mac users, who don't have
  it); and 5) saving and deleting user presets. ([#91][i91])
- Panic the synth if it outputs NaN or Inf, and handle these more gracefully in
  the loudness and peak detector. ([#210][i210])
- More presets from Reaby, and all new and existing presets were normalized
  roughly to -12 dBFS true peak. ([#211][i211])

### Fixed
- Occasional NaNs in the Trisaw oscillator when the color was 0 in the Go VM.
- The tracker thought that "sync" unit pops the value from stack, even if the VM
  did not, resulting it claiming errors in patches that worked once compiled.

### Changed
- Tracker model supports now enum-style values, which are integers that have a
  name associated with them. These enums are used to display menus where you
  select one of the options, for example in the MIDI menu to choose one of the
  ports; a context menu in to choose which instrument triggers the oscilloscope;
  and a context menu to choose the weighting type in the loudness detector.
- The song panel can scroll if all the widgets don't fit into it
- The provided MacOS executables are now arm64, which means the x86 native
  synths are not compiled in.

## [0.5.0]
### BREAKING CHANGES
- BREAKING CHANGE: always first modulate delay time, then apply notetracking. In
  a delay unit, modulation adds to the delay time, while note tracking
  multiplies it with a multiplier dependent on the note. The order of these
  operations was different in the Go VM vs. x86 VM & WebAssembly VM. In the Go
  VM, it first modulated, and then applied the note tracking multiplication. In
  the two assembly VMs, it first applied the note tracking and then modulated.
  Of these two behaviours, the Go VM behaviour made more sense: if you make a
  vibrato of +-50 cents for C4, you probably want a vibrato of +-50 cents for C6
  also. Thus, first modulating and then applying the note tracking
  multiplication is now the behaviour accross all VMs.
- BREAKING CHANGE: the negbandpass and neghighpass parameters of the filter unit
  were removed. Setting bandpass or highpass to -1 achieves now the same end
  result. Setting both negbandpass and bandpass to 1 was previously a no-op. Old
  patch and instrument files are converted to the new format when loaded, but
  newer Sointu files should not be compiled with an old version of
  sointu-compile.  

### Added
- Signal rail that visualizes what happens in the stack, shown on the left side
  of each unit in the rack.
- The parameters are now displayed in a grid as knobs, with units of the
  instrument going from the top to the bottom. Bezier lines are used to indicate
  which sends modulate which ports. ([#173][i173])
- Tabbing works more consistently, with widgets placed in a "tree", and plain
  Tab moves to the next widget on the same level or more shallow in the tree,
  while ctrl-Tab moves to next widget, regardless of its depth. This allows the
  user to quickly move between different panels, but also tabbing into every
  tiny widget if needed. Shift-* tab backwards.
- Help menu, with a menu item to show the license in a dialog, and also menu
  items to open manual, Github Discussions & Github Issues in a browser
  ([#196][i196])
- Show CPU load percentage in the song panel ([#192][i192])
- Theme can be user configured, in theme.yml. This theme.yml should be placed in
  the usual sointu config directory (i.e.
  `os.UserConfigDir()/sointu/theme.yml`). See
  [theme.yml](tracker/gioui/theme.yml) for the default theme, and
  [theme.go](tracker/gioui/theme.go) for what can be changed.
- Ctrl + scroll wheel adjusts the global scaling of the GUI ([#153][i153])
- The loudness detection supports LUFS, A-weighting, C-weighting or
  RMS-weighting, and peak detection supports true peak or sample peak detection.
  The loudness and peak values are displayed in the song panel ([#186][i186])
- Oscilloscope to visualize the outputted waveform ([#61][i61])
- Toggle button to keep instruments and tracks linked, and buttons to split
  instruments and tracks with more than 1 voice into parallel ones
  ([#163][i163], [#157][i157])
- Mute and solo toggles for instruments ([#168][i168])
- Many units (e.g. envelopes, oscillators and compressors) display values dB
- Dragging mouse to select rectangles in the tables
- The standalone tracker can open a MIDI port for receiving MIDI notes
  ([#166][i166])
- The note editor has a button to allow entering notes by MIDI. ([#170][i170])
- Units can have comments, to make it easier to distinguish between units of
  same type within an instrument and to use these as subsection titles.
  ([#114][i114])
- A toggle button for copying non-unique patterns before editing. When enabled
  and if the pattern is used in multiple places, the pattern is copied first.
  ([#77][i77])
- User can define own keybindings in `os.UserConfigDir()/sointu/keybindings.yml`
  ([#94][i94], [#151][i151])
- User can define preferred window size in
  `os.UserConfigDir()/sointu/preferences.yml` ([#184][i184])
- A small number above the instrument name identifies the MIDI channel /
  instrument number, with numbering starting from 1 ([#154][i154])
- The filter unit frequency parameter is displayed in Hz, corresponding roughly
  to the resonant frequency of the filter ([#158][i158])
- Include version info in the binaries, as given be `git describe`. This version
  info is shown as a label in the tracker and can be checked with `-v` flag in
  the command line tools.
- Performance improvement: values needed by the UI that are derived from the
  score or patch are cached when score or patch changes, so they don't have to
  be computed every draw. ([#176][i176])

### Fixed
- Tooltips will be hidden after certain amount of time has passed, to ensure
  that the tooltips don't stay around ([#141][i141])
- Loading instrument forgot to close the file (in model.ReadInstrument)
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
- Pressing `a` or `1` when editing note values in hex mode created a note off
  line ([#162][i162])
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
- sointu-play allows choosing between the synths, assuming it was compiled with
  `-tags=native`
- Most buttons never gain focus, so that clicking a button does not stop
  whatever the user was currently doing and so that the user does not
  accidentally trigger the buttons by having them focused and e.g. hitting space
  ([#156][i156])

### Changed
- When saving instrument to a file, the instrument name is not saved to the name
  field, as Sointu will anyway use the filename as the instrument's name when it
  is loaded.
- Native version of the tracker/VSTi was removed. Instead, you can change
  between the two versions of the synth on the fly, by clicking on the "Synth"
  option under the CPU group in the song panel ([#200][i200])
- Send amount defaults to 64 = 0.0 ([#178][i178])
- The maximum number of delaylines in the native synth was increased to 128,
  with slight increase in memory usage ([#155][i155])
- The numeric updown widget has a new appearance.
- The draggable UI splitters snap more controllably to the window edges.
- New & better presets, organized by their type to subfolders (thanks Reaby!)
  ([#136][i136])
- Presets get their name by concatenating their subdirectory path (with path
  separators replaced with spaces) to their filename
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

[Unreleased]: https://github.com/vsariola/sointu/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/vsariola/sointu/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/vsariola/sointu/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/vsariola/sointu/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/vsariola/sointu/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/vsariola/sointu/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/vsariola/sointu/compare/4klang-3.11...v0.1.0
[i61]: https://github.com/vsariola/sointu/issues/61
[i65]: https://github.com/vsariola/sointu/issues/65
[i67]: https://github.com/vsariola/sointu/issues/67
[i68]: https://github.com/vsariola/sointu/issues/68
[i77]: https://github.com/vsariola/sointu/issues/77
[i91]: https://github.com/vsariola/sointu/issues/91
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
[i136]: https://github.com/vsariola/sointu/issues/136
[i139]: https://github.com/vsariola/sointu/issues/139
[i141]: https://github.com/vsariola/sointu/issues/141
[i142]: https://github.com/vsariola/sointu/issues/142
[i144]: https://github.com/vsariola/sointu/issues/144
[i145]: https://github.com/vsariola/sointu/issues/145
[i146]: https://github.com/vsariola/sointu/issues/146
[i148]: https://github.com/vsariola/sointu/issues/148
[i149]: https://github.com/vsariola/sointu/issues/149
[i150]: https://github.com/vsariola/sointu/issues/150
[i151]: https://github.com/vsariola/sointu/issues/151
[i153]: https://github.com/vsariola/sointu/issues/153
[i154]: https://github.com/vsariola/sointu/issues/154
[i155]: https://github.com/vsariola/sointu/issues/155
[i156]: https://github.com/vsariola/sointu/issues/156
[i157]: https://github.com/vsariola/sointu/issues/157
[i158]: https://github.com/vsariola/sointu/issues/158
[i160]: https://github.com/vsariola/sointu/issues/160
[i162]: https://github.com/vsariola/sointu/issues/162
[i163]: https://github.com/vsariola/sointu/issues/163
[i166]: https://github.com/vsariola/sointu/issues/166
[i168]: https://github.com/vsariola/sointu/issues/168
[i170]: https://github.com/vsariola/sointu/issues/170
[i173]: https://github.com/vsariola/sointu/issues/173
[i176]: https://github.com/vsariola/sointu/issues/176
[i178]: https://github.com/vsariola/sointu/issues/178
[i184]: https://github.com/vsariola/sointu/issues/184
[i186]: https://github.com/vsariola/sointu/issues/186
[i192]: https://github.com/vsariola/sointu/issues/192
[i196]: https://github.com/vsariola/sointu/issues/196
[i199]: https://github.com/vsariola/sointu/issues/199
[i200]: https://github.com/vsariola/sointu/issues/200
[i210]: https://github.com/vsariola/sointu/issues/210
[i211]: https://github.com/vsariola/sointu/issues/211
