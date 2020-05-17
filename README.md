# Sointu
A cross-platform modular software synthesizer for small intros, forked from
[4klang](https://github.com/hzdgopher/4klang).

Summary
-------

Sointu is work-in-progress. It is a fork and an evolution of [4klang]( 
https://github.com/hzdgopher/4klang), a modular software synthesizer intended 
to easily produce music for 4k intros-small executables with a maximum 
filesize of 4096 bytes containing realtime audio and visuals. Like 4klang, the
 sound is produced by a virtual machine that executes small bytecode to 
produce the audio; however, by now the internal virtual machine has been 
heavily rewritten and extended to make the code more maintainable, possibly 
even saving some bytes in the process. 

New features since fork
-----------------------
  - **Per instrument polyphonism**. An instrument has the possibility to 
    have any number of voices, meaning in practice that multiple voices can
    reuse the same opcodes. Done, see [here](tests/test_polyphony.asm) for an 
    example and [here](src/opcodes/flowcontrol.asm) for the implementation. The 
    maximum total number of voices will be 32: you can have 32 monophonic 
    instruments or any combination of polyphonic instruments adding up to 32. 
  - **Any number of voices per track**. For example, a polyphonic instrument of 
    3 voices can be triggered by 3 parallel tracks, to produce chords. But one 
    track can also trigger 3 voices, for example when using arpeggio. A track 
    can even trigger 2 voices of different instruments, alternating between 
    these two; maybe useful for example as an easy way to alternate between an 
    open and a closed hihat.
  - **Easily extensible**. Instead of %ifdef hell, the primary extension
    mechanism will be through new opcodes for the virtual machine. Only the
    opcodes actually used in a song are compiled into the virtual machine. The
    goal is to try to write the code so that if two similar opcodes are used,
    the common code in both is reused by moving it to a function.
  - **Take the macro languge to its logical conclusion**. Only the patch
    definition should be needed; all the %define USE_SOMETHING will be
    defined automatically by the macros. Furthermore, only the opcodes needed
    are compiled into the program. Done, see for example
    [this test](tests/test_oscillat_trisaw.asm)! This has the nice implication that,
    in future, there will be no need for binary format to save patches: the .asm
    is easy enough to be imported into / exported from the GUI. Being a text
    format, the .asm based patch definitions play nicely with source control.
  - **Harmonized support for stereo signals**. Every opcode supports a stereo 
    variant: the stereo bit is hidden in the least significant bit of the 
    command stream and passed in carry to the opcode. This has several nice 
    advantages: 1) the opcodes that don't need any parameters do not need an 
    entire byte in the value stream to define whether it is stereo; 2) stereo 
    variants of opcodes can be implemented rather efficiently; in many cases, 
    the extra cost of stereo variant is only 7 bytes, of which 4 are zeros, so
    should compress quite nicely. 3) Since stereo opcodes usually follow stereo
    opcodes (and mono opcodes follow mono opcodes), the stereo bits of the
    command bytes will be highly correlated and if crinkler or any other
    modeling compressor is doing its job, that should make them highly
    predictable i.e. highly compressably. Done.
  - **Test-driven development**. Given that 4klang was already a mature project, 
    the first thing actually implemented was a set of regression tests to avoid 
    breaking everything beyond any hope of repair. Done, using CTest.
  - **New units**. Bit-crusher, gain, inverse gain, clip, modulate bpm
    (proper triplets!), compressor (can be used for side-chaining)... As 
    always, if you don't use them, they won't be compiled into the code.
  - **Arbitrary signal routing**. SEND (used to be called FST) opcode normally 
    sends the signal as a modulation to another opcode. But with the new 
    RECEIVE opcode, you just receive the plain signal there. So you can connect
    signals in an arbitrary way. Actually, 4klang could already do this but in
    a very awkward way: it had FLD (load value) opcode that could be modulated;
    FLD 0 with modulation basically achieved what RECEIVE does, except that
    RECEIVE can also handle stereo signals.
  - **Pattern length does not have to be a power of 2**.

Future goals
------------

  - **Cross-platform support for win / mac / linux**. The build is already based 
    on CMake and compiles on Windows. Cross-platform NASM/YASM macros have been
    drafted and remain to be tested. Once the project is more mature, I will 
    try compiling on other platforms.
  - **Find a more general solution for skipping opcodes / early outs**. It's 
    probably a new opcode "skip" that skips from the opcode to the next out in
    case the signal entering skip and the signal leaving out are both close to
    zero.
  - **Even more opcodes**. Maybe an equalizer? DC-offset removal?
  - **Support for 64-bit targets**.
  - **Browser-based GUI and MIDI instrument**. Modern browsers support WebMIDI,
     WebAudio and, most importantly, they are cross-platform and come installed
     on pretty much any computer. The only thing needed is to be able to
     communicate with the platform specific synth; for this, the best
     option seems to be to run the synth inside a tiny websocket server that
     receives messages from browser and streams the audio to the  browser. 
     The feasibility of the approach is proven (localhost websocket calls
     have 1 ms range of latency), but nothing more is done yet.

Nice-to-have ideas
------------------

  - **Sample import from gm.dls**. This is Windows only, but implementing it 
    should be easy and the potential payoffs pretty high for Windows users, so 
    it is a nice prospect.
  - **Tracker**. If the list of primary goals is ever exhausted, a browser-based
    tracker would be nice to take advantage of all the features.

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
    maintainable way to make the project into DAW plugin, I may reconsider.

Design philosophy
-----------------

  - Try to avoid %ifdef hell as much as possible. If needed, try to include all
    code toggled by a define in one block.
  - Instead of prematurely adding %ifdef toggles to optimize away unused 
    features, start with the most advanced featureset and see if you can 
    implement it in a generalized way. For example, all the modulations are 
    now added into the values when they are converted from integers, in a
    standardized way. This got rid of most of the %ifdefs in 4klang. Also, with
    no %ifdefs cluttering the view, many opportunities to shave away
    instructions became apparent. Also, by making the most advanced synth
    cheaply available to the scene, we promote better music in future 4ks :)
  - Size first, speed second. Speed will only considered if the situation 
    becomes untolerable.
  - Benchmark optimizations. Compression results are sometimes slightly 
    nonintuitive so alternative implementations should always be benchmarked
    e.g. by compiling and linking a real-world song with [Leviathan](https://github.com/armak/Leviathan-2.0)
    and observing how the optimizations
    affect the byte size.

Background and history
----------------------

[4klang](https://github.com/hzdgopher/4klang) development was started in 2007 
by Dominik Ries (gopher) and Paul Kraus (pOWL) of Alcatraz. The [write-up](
http://zine.bitfellas.org/article.php?zine=14&id=35) will still be helpful for
 anyone looking to understand how 4klang and Sointu use the FPU stack to 
manipulate the signals. Since then, 4klang has been used in countless of scene
 productions and people use it even today.

However, 4klang is pretty deep in the [%ifdef hell](https://www.cqse.eu/en/blog/living-in-the-ifdef-hell/), 
and the polyphonism was never implemented in a very well engineered way (you 
can have exactly 2 voices per instrument if you enable it). Also, reading 
through the code, I spotted several avenues to squeeze away more bytes. These 
observations triggered project Sointu. That, and I just wanted to learn x86 
assembly, and needed a real-world project to work on. 

Credits
-------

The original 4klang was developed by Dominik Ries ([gopher](https://github.com/hzdgopher/4klang)) and Paul Kraus 
(pOWL) of Alcatraz.

Sointu was initiated by Veikko Sariola (pestis/bC!).

PoroCYon's [4klang fork](https://github.com/PoroCYon/4klang) inspired the macros
to better support cross-platform asm.