%ifndef SOINTU_INC
%define SOINTU_INC

; Following defines have to be defined before this include:
; MAX_TRACKS            e.g.         %define MAX_TRACKS 10
; BPM                   e.g.         %define BPM 100
; MAX_PATTERNS          e.g.         %define MAX_PATTERNS 1
; MAX_VOICES            e.g.         %define MAX_VOICES 4 <- e.g. 4 instruments or 1 polyphonic instrument with 4 voices
;
; Optionally:
; PATTERN_SIZE          e.g.         %define PATTERN_SIZE 16 <- this is the default

%macro EXPORT 1
    global %1
    %1
%endmacro

%ifidn __OUTPUT_FORMAT__,win32
    ; on win32, function f with n parameters is mangled as "_f@n"
    %define MANGLE_FUNC(f,n) _ %+ f %+ @ %+ n
    %define WIN_OR_MAC
    %assign BITS 32
    ; On windows and mac, data label d is mangled as "_d"
    %define MANGLE_DATA(d) _ %+ d
%endif

%ifidn __OUTPUT_FORMAT__,win64
    ; on win32, function f with n parameters is mangled as "_f@n"
    %define MANGLE_FUNC(f,n) f
    %define WIN_OR_MAC
    %assign BITS 64
    ; On windows and mac, data label d is mangled as "_d"
    %define MANGLE_DATA(d) d
%endif

%ifidn __OUTPUT_FORMAT__,elf32
    ; on linux, function f with n parameters is mangled as "f"
    %define MANGLE_FUNC(f,n) f
    ; On linux, data label d is mangled as "d"
    %define MANGLE_DATA(d) d
%endif

%ifidn __OUTPUT_FORMAT__,macho32
    ; on mac, function f with x parameters is mangled as "_f"
    %define MANGLE_FUNC(f,n) _f
    %define WIN_OR_MAC
    ; On windows and mac, data label d is mangled as "_d"
    %define MANGLE_DATA(d) _ %+ d
%endif

%ifdef WIN_OR_MAC
    ; Windows has crinkler so one may USE_SECTIONS to put everything in custom sections to aid crinkler.
    ; Maybe mac users need it too
    %ifdef USE_SECTIONS
        %define SECT_BSS(n) section . %+ n bss align=1
        %define SECT_DATA(n) section . %+ n data align=1
        %define SECT_TEXT(n) section . %+ n code align=1
    %else
        %define SECT_BSS(n) section .bss align=1
        %define SECT_DATA(n) section .data align=1
        %define SECT_TEXT(n) section .code align=1
    %endif
%else
    ; Linux
    %ifdef USE_SECTIONS
        %define SECT_BSS(n) section .bss. %+ n nobits alloc noexec write align=1
        %define SECT_DATA(n) section .data. %+ n progbits alloc noexec write align=1
        %define SECT_TEXT(n) section .text. %+ n progbits alloc exec nowrite align=1
    %else
        %define SECT_BSS(n) section .bss. nobits alloc noexec write align=1
        %define SECT_DATA(n) section .data. progbits alloc noexec write align=1
        %define SECT_TEXT(n) section .text. progbits alloc exec nowrite align=1
    %endif
%endif

%ifdef SU_USE_ALL
    ; SU_USE_ALL is convenience way to enable almost all features of the synth, for vsti plugins and such which
    ; do not have any size constraints. However, SU_USE_ALL should only enable features that absolutely do not
    ; change the functioning of the synth in any way, just add features. Clipping, 16 bit output etc. should still
    ; be enabled only whent they are actually needed

    ; Things that are NOT defined by SU_USE_ALL
    ;%define SU_USE_16BIT_OUTPUT         ; // removing this will output to 32bit floating point buffer
    ;%define SU_USE_GROOVE_PATTERN       ; // removing this skips groove pattern code
    ;%define SU_USE_ENVELOPE_RECORDINGS  ; // removing this skips envelope recording code
    ;%define SU_USE_NOTE_RECORDINGS      ; // removing this skips note recording code
    ;%define SU_USE_UNDENORMALIZE        ; // removing this skips denormalization code in the units
    ;%define SU_CLIP_OUTPUT              ; // removing this skips clipping code for the final output

    %define SU_USE_DST                    ; // removing this will skip DST unit
    %define SU_USE_DLL                    ; // removing this will   skip DLL unit
    %define SU_USE_PAN                    ; // removing this will   skip PAN unit
    %define SU_USE_GLOBAL_DLL             ; // removing this will   skip global dll processing
    %define SU_USE_FSTG                   ; // removing this will   skip global store unit
    %define SU_USE_FLD                    ; // removing this will   skip float load unit
    %define SU_USE_GLITCH                 ; // removing this will   skip GLITCH unit
    %define SU_USE_ENV_CHECK              ; // removing this skips checks   if processing is needed
    %define SU_USE_VCO_CHECK              ; // removing this skips checks   if processing is needed
    %define SU_USE_VCO_PHASE_OFFSET       ; // removing this will   skip initial phase offset code
    %define SU_USE_VCO_SHAPE                ; // removing this skips waveshaping code
    %define SU_USE_VCO_GATE             ; // removing this skips gate code
    %define SU_USE_VCO_MOD_FM               ; // removing this skips frequency modulation code
    %define SU_USE_VCO_MOD_DM               ; // removing this skips detune modulation code
    %define SU_USE_VCO_STEREO               ; // removing this skips stereo code
    %define SU_USE_VCF_CHECK                ; // removing this skips checks if processing is needed
    %define SU_USE_VCF_HIGH             ; // removing this skips code for high output
    %define SU_USE_VCF_BAND             ; // removing this skips code for band output
    %define SU_USE_VCF_PEAK             ; // removing this skips code for peak output
    %define SU_USE_VCF_STEREO               ; // removing this skips code for stereo filter output
    %define SU_USE_DST_CHECK                ; // removing this skips checks if processing is needed
    %define SU_USE_DST_SH                   ; // removing this skips sample and hold code
    %define SU_USE_DST_STEREO             ; // removing this skips stereo processing
    %define SU_USE_DLL_NOTE_SYNC          ; // removing this will   skip delay length adjusting code (karplus strong)
    %define SU_USE_DLL_CHORUS             ; // removing this will   skip delay chorus/flanger code
    %define SU_USE_DLL_CHORUS_CLAMP       ; // removing this will skip chorus lfo phase clamping
    %define SU_USE_DLL_DAMP             ; // removing this will skip dll damping code
    %define SU_USE_DLL_DC_FILTER            ; // removing this will skip dll dc offset removal code
    %define SU_USE_FSTG_CHECK               ; // removing this skips checks if processing is needed
    %define SU_USE_WAVESHAPER_CLIP      ; // removing this will skip clipping code
%endif

%ifdef SU_USE_VCO_SHAPE
    %define INCLUDE_WAVESHAPER
%endif
%ifdef SU_USE_DST
    %define INCLUDE_WAVESHAPER
%endif
%ifdef SU_USE_16BIT_OUTPUT
    %define SU_INCLUDE_CLIP
%endif

;%include "opcodes/flowcontrol.inc"

%assign CUR_ID          2
%define CMDS              ; CMDS is empty at first, no commands defined
%define OPCODES         MANGLE_FUNC(su_op_advance,0),
%define NUMPARAMS       0,
%define SU_ADVANCE_ID   0
%define MONO            0
%define STEREO          1

%include "opcodes/flowcontrol.inc"
%include "opcodes/arithmetic.inc"
%include "opcodes/effects.inc"
%include "opcodes/sources.inc"
%include "opcodes/sinks.inc"

; //----------------------------------------------------------------------------------------
; // synth defines
; //----------------------------------------------------------------------------------------

%define MAX_DELAY           65536
%assign MAX_UNITS_SHIFT     6
%assign MAX_UNITS           ((1 << MAX_UNITS_SHIFT)-1) ; this is carefully chosen to align su_unit to 2^n boundary
%define ABSOLUTE_MAX_VOICES 32

%ifndef SAMPLE_RATE
%define SAMPLE_RATE 44100
%endif

%ifndef HLD
%define HLD 1
%endif

%define TOTAL_ROWS (MAX_PATTERNS*PATTERN_SIZE)
%define SAMPLES_PER_ROW (SAMPLE_RATE*4*60/(BPM*16))
%define DEF_LFO_NORMALIZE 0.000038
%define MAX_SAMPLES (SAMPLES_PER_ROW*TOTAL_ROWS)

%macro SU_BEGIN_PATCH 0
    SECT_DATA(params)

    EXPORT MANGLE_DATA(su_params)
%endmacro

%macro SU_END_PATCH 0 ; After the patch is finished, saves the accumulated commands
    SECT_DATA(sucomnds)

    EXPORT MANGLE_DATA(su_commands)
        db      CMDS
%endmacro

%define CONCATENATE(x,y) x %+ y
%define POLYPHONY_BITMASK 0
%assign MAX_VOICES 0
%assign MAX_TRACKS 0
%macro SU_BEGIN_INSTRUMENT 1
    ; increment MAX_VOICES equal to %1 and construct the POLYPHONY_BITMASK so that
    ; for every except the last, the bit is on
    %rep  %1-1
        %assign POLYPHONY_BITMASK POLYPHONY_BITMASK + (1 << MAX_VOICES)
        %assign MAX_VOICES MAX_VOICES + 1
    %endrep
    %assign MAX_VOICES MAX_VOICES + 1 ; the last voice increment, without adding bit mask
    %if MAX_VOICES > 32
        %error Error: cannot have more than 32 voices!
    %endif
    %if %1 > 1
        %define INCLUDE_POLYPHONY
    %endif
%endmacro

%define VOICES(val)     val
%define TRACKS(val)     val

%macro SU_END_INSTRUMENT 0
    %xdefine CMDS CMDS SU_ADVANCE_ID,
%endmacro

%assign PATTERN_LENGTH -1
%macro SU_BEGIN_PATTERNS 0
    SECT_DATA(supatrns)

    EXPORT MANGLE_DATA(su_patterns)
    %define USE_PLAYER
%endmacro

%define SU_END_PATTERNS

%assign PATTERN_SIZE -1
%macro PATTERN 1-*
    %rep  %0
        db      %1
        %rotate 1
    %endrep
    %if %0 >= 256
        %error 'Pattern size should be < 256'
    %endif
    %if PATTERN_SIZE == -1
        %assign PATTERN_SIZE %0
    %else
        %if %0 != PATTERN_SIZE
            %error 'All patterns should have the same length!'
        %endif
    %endif
%endmacro

%macro SU_BEGIN_TRACKS 0
    SECT_DATA(sutracks)

    EXPORT MANGLE_DATA(su_tracks)
    %define USE_PLAYER
%endmacro

%assign MAX_PATTERNS -1
%assign MAX_TRACKS 0
%assign VOICETRACK_BITMASK 0
%assign VOICETRACK_COUNT 0
%macro TRACK 2-* ; first param number of voices, rest are the patterns
    %rep  %0-1
        db      %2
        %rotate 1
    %endrep
    %rotate 1
    %if MAX_PATTERNS == -1
        %assign MAX_PATTERNS %0-1
    %else
        %if %0-1 != MAX_PATTERNS
            %error 'All tracks should have same number of patterns!'
        %endif
    %endif
    %assign MAX_TRACKS MAX_TRACKS + 1
    %if MAX_TRACKS > 32
        %error Error: cannot have more than 32 tracks!
    %endif

    ; increment MAX_TRACKS equal to %2 and construct the CHORD_BITMASK so that
    ; for every track except the last track of an instrument, the bit is on
    %rep  %1-1
        %assign VOICETRACK_BITMASK VOICETRACK_BITMASK + (1 << VOICETRACK_COUNT)
        %assign VOICETRACK_COUNT VOICETRACK_COUNT + 1
    %endrep
    %assign VOICETRACK_COUNT VOICETRACK_COUNT + 1 ; the last voice increment, without adding bit mask
    %if VOICETRACK_COUNT > 32
        %error Error: cannot have more than a total of 32 voices assigned to tracks.
    %endif
    %if %1 > 1
        %define INCLUDE_MULTIVOICE_TRACKS
    %endif
%endmacro

%define SU_END_TRACKS

; //----------------------------------------------------------------------------------------
; // Unit struct
; //----------------------------------------------------------------------------------------
struc su_unit
    .state      resd    8
    .ports      resd    8
    .size
endstruc

; //----------------------------------------------------------------------------------------
; // Voice struct
; //----------------------------------------------------------------------------------------
struc su_voice
    .note       resd    1
    .release    resd    1
    .reserved   resb    (su_unit.size - 8) ; this is done to so the whole voice is 2^n long, see polyphonic player
    .workspace  resb    MAX_UNITS * su_unit.size
    .size
endstruc
; //----------------------------------------------------------------------------------------
; // Synth struct
; //----------------------------------------------------------------------------------------
struc su_synth
    .left       resd    1
    .right      resd    1
    .voices     resb    ABSOLUTE_MAX_VOICES * su_voice.size
    .size
endstruc

%endif ; SOINTU_INC
