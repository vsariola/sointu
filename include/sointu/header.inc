%ifndef SOINTU_INC
%define SOINTU_INC

; You will have to define a BPM for your song, e.g.
; %define BPM 100

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

%ifidn __OUTPUT_FORMAT__,elf
    ; on linux, function f with n parameters is mangled as "f"
    %define MANGLE_FUNC(f,n) f
    ; On linux, data label d is mangled as "d"
    %define MANGLE_DATA(d) d
    %assign BITS 32
%endif

%ifidn __OUTPUT_FORMAT__,elf64
    %define MANGLE_FUNC(f,n) f
    %define MANGLE_DATA(d) d
    %assign BITS 64
%endif

%ifidn __OUTPUT_FORMAT__,macho32
    ; on mac, function f with x parameters is mangled as "_f"
    %define MANGLE_FUNC(f,n) _f
    %define WIN_OR_MAC
    ; On windows and mac, data label d is mangled as "_d"
    %define MANGLE_DATA(d) _ %+ d
%endif

%ifdef WIN_OR_MAC
    ; Windows has crinkler so one may put everything in custom sections to aid crinkler.
    ; Maybe mac users need it too
    %ifndef DISABLE_SECTIONS
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
    %ifndef DISABLE_SECTIONS
        %define SECT_BSS(n) section .bss. %+ n nobits alloc noexec write align=1
        %define SECT_DATA(n) section .data. %+ n progbits alloc noexec write align=1
        %define SECT_TEXT(n) section .text. %+ n progbits alloc exec nowrite align=1
    %else
        %define SECT_BSS(n) section .bss. nobits alloc noexec write align=1
        %define SECT_DATA(n) section .data. progbits alloc noexec write align=1
        %define SECT_TEXT(n) section .text. progbits alloc exec nowrite align=1
    %endif
%endif

%ifdef SU_USE_16BIT_OUTPUT
    %define SU_INCLUDE_CLIP
%endif

%assign CUR_ID          2
%define CMDS              ; CMDS is empty at first, no commands defined
%define OPCODES         MANGLE_FUNC(su_op_advance,0),
%define NUMPARAMS       0,
%define SU_ADVANCE_ID   0
%define MONO            0
%define STEREO          1

section .text ; yasm throws section redeclaration warnings if strucs are defined without a plain .text section

%include "sointu/flowcontrol_header.inc"
%include "sointu/arithmetic_header.inc"
%include "sointu/effects_header.inc"
%include "sointu/sources_header.inc"
%include "sointu/sinks_header.inc"

;-------------------------------------------------------------------------------
;   synth defines
;-------------------------------------------------------------------------------

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
%define MAX_SAMPLES (SAMPLES_PER_ROW*TOTAL_ROWS)

%macro BEGIN_PATCH 0
    SECT_DATA(params)

    EXPORT MANGLE_DATA(su_params)
%endmacro

%macro END_PATCH 0 ; After the patch is finished, saves the accumulated commands
    SECT_DATA(sucomnds)

    EXPORT MANGLE_DATA(su_commands)
        db      CMDS
%endmacro

%define POLYPHONY_BITMASK 0
%assign MAX_VOICES 0
%assign MAX_TRACKS 0
%macro BEGIN_INSTRUMENT 1
    ; increment MAX_VOICES equal to %1 and construct the POLYPHONY_BITMASK so that
    ; for every except the last, the bit is on
    %rep  %1-1
        %assign POLYPHONY_BITMASK (POLYPHONY_BITMASK << 1) + 1
        %assign MAX_VOICES MAX_VOICES + 1
    %endrep
    %assign POLYPHONY_BITMASK (POLYPHONY_BITMASK << 1)
    %assign MAX_VOICES MAX_VOICES + 1 ; the last voice increment, without adding one bit to the mask
    %if MAX_VOICES > 32
        %error Error: cannot have more than 32 voices!
    %endif
    %if %1 > 1
        %define INCLUDE_POLYPHONY
    %endif
%endmacro

%define VOICES(val)     val
%define TRACKS(val)     val

%macro END_INSTRUMENT 0
    %xdefine CMDS CMDS SU_ADVANCE_ID,
%endmacro

%assign PATTERN_LENGTH -1
%macro BEGIN_PATTERNS 0
    SECT_DATA(supatrns)

    EXPORT MANGLE_DATA(su_patterns)
    %define USE_PLAYER
%endmacro

%define END_PATTERNS

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

%macro BEGIN_TRACKS 0
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

%define END_TRACKS

;-------------------------------------------------------------------------------
;   unit struct
;-------------------------------------------------------------------------------
struc su_unit
    .state      resd    8
    .ports      resd    8
    .size
endstruc

;-------------------------------------------------------------------------------
;   voice struct
;-------------------------------------------------------------------------------
struc su_voice
    .note       resd    1
    .release    resd    1
    .inputs     resd    8
    .reserved   resd    6 ; this is done to so the whole voice is 2^n long, see polyphonic player
    .workspace  resb    MAX_UNITS * su_unit.size
    .size
endstruc

;-------------------------------------------------------------------------------
;   synthworkspace struct
;-------------------------------------------------------------------------------
struc su_synthworkspace
    .curvoices  resb    32      ; these are used by the multitrack player to store which voice is playing on which track
    .left       resd    1
    .right      resd    1
    .aux        resd    6       ; 3 auxiliary signals
    .voices     resb    ABSOLUTE_MAX_VOICES * su_voice.size
    .size
endstruc

%endif ; SOINTU_INC
