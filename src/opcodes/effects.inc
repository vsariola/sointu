;-------------------------------------------------------------------------------
;   Filter (LOWPASS, BANDPASS...) effect related defines
;-------------------------------------------------------------------------------
%assign FILTER_ID -1

%macro USE_FILTER 0
    %if FILTER_ID == -1
        %assign FILTER_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_filter,0),
        %xdefine NUMPARAMS NUMPARAMS 2,
    %endif
%endmacro

%define LOWPASS     0x40
%define BANDPASS    0x20
%define HIGHPASS    0x10
%define NEGBANDPASS 0x08
%define NEGHIGHPASS 0x04

%macro  SU_FILTER 4
    db  %2
    db  %3
    db  %4
    USE_FILTER
    %xdefine CMDS CMDS FILTER_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_FILTER
    %endif
    %if (%4) & LOWPASS == LOWPASS
        %define INCLUDE_LOWPASS
    %endif
    %if (%4) & BANDPASS == BANDPASS
        %define INCLUDE_BANDPASS
    %endif
    %if (%4) & HIGHPASS == HIGHPASS
        %define INCLUDE_HIGHPASS
    %endif
    %if (%4) & NEGBANDPASS == NEGBANDPASS
        %define INCLUDE_NEGBANDPASS
    %endif
    %if (%4) & NEGHIGHPASS == NEGHIGHPASS
        %define INCLUDE_NEGHIGHPASS
    %endif
%endmacro

%define FREQUENCY(val)  val
%define RESONANCE(val)  val
%define FLAGS(val)      val

struc   su_filter_ports
    .freq       resd    1
    .res        resd    1
    .params
endstruc

struc   su_filter_wrk
    .low        resd    1
    .high       resd    1
    .band       resd    1
    .size
endstruc

;-------------------------------------------------------------------------------
;   PAN effect related defines
;-------------------------------------------------------------------------------
%assign PAN_ID -1

%macro USE_PAN 0
    %if PAN_ID == -1
        %assign PAN_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_pan,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro  SU_PAN 2
    db  %2
    USE_PAN
    %xdefine CMDS CMDS PAN_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_PAN
    %endif
%endmacro

%define PANNING(val)    val

struc   su_pan_ports
    .panning    resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   DISTORT effect related defines
;-------------------------------------------------------------------------------
%assign DISTORT_ID -1

%macro USE_DISTORT 0
    %if DISTORT_ID == -1
        %assign DISTORT_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_distort,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro SU_DISTORT 2
    db  %2
    USE_DISTORT
    %xdefine CMDS CMDS DISTORT_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_DISTORT
    %endif
%endmacro

%define DRIVE(val)      val

struc   su_distort_ports
    .drive      resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   HOLD effect related defines
;-------------------------------------------------------------------------------
%assign HOLD_ID -1

%macro USE_HOLD 0
    %if HOLD_ID == -1
        %assign HOLD_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_hold,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro SU_HOLD 2
    db  %2
    USE_HOLD
    %xdefine CMDS CMDS HOLD_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_HOLD
    %endif
%endmacro

%define HOLDFREQ(val) val

struc su_hold_ports
    .freq       resd    1
    .params
endstruc

struc su_hold_wrk
    .phase      resd    1
    .holdval    resd    1
endstruc

;-------------------------------------------------------------------------------
;   CRUSH effect related defines
;-------------------------------------------------------------------------------
%assign CRUSH_ID -1

%macro USE_CRUSH 0
    %if CRUSH_ID == -1
        %assign CRUSH_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_crush,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro SU_CRUSH 2
    db  %2
    USE_CRUSH
    %xdefine CMDS CMDS CRUSH_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_CRUSH
    %endif
%endmacro

%define RESOLUTION(val) val

struc su_crush_ports
    .resolution     resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   GAIN effect related defines
;-------------------------------------------------------------------------------
%assign GAIN_ID -1

%macro USE_GAIN 0
    %if GAIN_ID == -1
        %assign GAIN_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_gain,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro SU_GAIN 2
    db  %2
    USE_GAIN
    %xdefine CMDS CMDS GAIN_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_GAIN
    %endif
%endmacro

%define GAIN(val) val

struc su_gain_ports
    .gain       resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   INVGAIN effect related defines
;-------------------------------------------------------------------------------
%assign INVGAIN_ID -1

%macro USE_INVGAIN 0
    %if INVGAIN_ID == -1
        %assign INVGAIN_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_invgain,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro SU_INVGAIN 2
    db  %2
    USE_INVGAIN
    %xdefine CMDS CMDS INVGAIN_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_INVGAIN
    %endif
%endmacro

%define INVGAIN(val) val

struc su_invgain_ports
    .invgain        resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   CLIP effect related defines
;-------------------------------------------------------------------------------
%assign CLIP_ID -1

%macro USE_CLIP 0
    %if CLIP_ID == -1
        %assign CLIP_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_clip,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_CLIP 1
    USE_CLIP
    %xdefine CMDS CMDS CLIP_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_CLIP
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   Delay effect related defines
;-------------------------------------------------------------------------------
%assign DELAY_ID -1
%macro USE_DELAY 0
    %if DELAY_ID == -1
        %assign DELAY_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_delay,0),
        %xdefine NUMPARAMS NUMPARAMS 4,
    %endif
%endmacro

%define MAX_DELAY 65536
%assign NUM_DELAY_LINES 0

%macro  SU_DELAY 7
    db  %2
    db  %3
    db  %4
    db  %5
    db  %6
    db  %7
    USE_DELAY
    %xdefine CMDS CMDS DELAY_ID + %1,
    %assign NUM_DELAY_LINES NUM_DELAY_LINES + %7 * (1+%1)
    %if %1 == STEREO
        %define INCLUDE_STEREO_DELAY
    %endif
    %if %6 == 0
        %define DELAY_NOTE_SYNC
    %endif
%endmacro

%macro SU_BEGIN_DELTIMES 0
    SECT_DATA(sudeltim)

    EXPORT MANGLE_DATA(su_delay_times)
        dw 0
%endmacro

%define SU_END_DELTIMES

%macro DELTIME 1-*
    %rep  %0
        dw      %1
        %rotate 1
    %endrep
%endmacro


%define PREGAIN(val)    val
%define DRY(val)        val
%define FEEDBACK(val)   val
%define DEPTH(val)      val
%define DAMP(val)       val
%define DELAY(val)      val
%define COUNT(val)      val

struc   su_delay_ports
    .pregain    resd    1
    .dry        resd    1
    .feedback   resd    1
    .damp       resd    1
    .freq       resd    1
    .ports
endstruc

struc   su_delayline_wrk
    .time       resd    1
    .filtstate  resd    1
    .dcin       resd    1
    .dcout      resd    1
    .buffer     resd    MAX_DELAY
    .size
endstruc
