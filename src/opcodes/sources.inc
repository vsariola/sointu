;-------------------------------------------------------------------------------
;   ENVELOPE structs
;-------------------------------------------------------------------------------
%assign ENVELOPE_ID -1
%macro USE_ENVELOPE 0
    %if ENVELOPE_ID == -1
        %assign ENVELOPE_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_envelope,0),
        %xdefine NUMPARAMS NUMPARAMS 5,
    %endif
%endmacro

%macro SU_ENVELOPE 6
    db  %2
    db  %3
    db  %4
    db  %5
    db  %6
    USE_ENVELOPE
    %xdefine CMDS CMDS ENVELOPE_ID+%1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_ENVELOPE
    %endif
%endmacro

%define ATTAC(val)      val
%define DECAY(val)      val
%define SUSTAIN(val)    val
%define RELEASE(val)    val
%define GAIN(val)       val

struc   su_env_ports
    .attac      resd    1
    .decay      resd    1
    .sustain    resd    1
    .release    resd    1
    .gain       resd    1
    .params
endstruc

struc   su_env_work
    .state      resd    1
    .level      resd    1
    .size
endstruc

%define ENV_STATE_ATTAC     0
%define ENV_STATE_DECAY     1
%define ENV_STATE_SUSTAIN   2
%define ENV_STATE_RELEASE   3
%define ENV_STATE_OFF       4

;-------------------------------------------------------------------------------
;   OSCILLAT structs
;-------------------------------------------------------------------------------
%assign OSCILLAT_ID -1
%macro USE_OSCILLAT 0
    %if OSCILLAT_ID == -1
        %assign OSCILLAT_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_oscillat,0),
        %xdefine NUMPARAMS NUMPARAMS 6,
    %endif
%endmacro

%define SAMPLE      0x80
%define SINE        0x40
%define TRISAW      0x20
%define PULSE       0x10
%define LFO         0x08
%define GATE        0x04
%define UNISON2     0x01
%define UNISON3     0x02 ; Warning, UNISON3 and UNISON4 do not work with gate at the moment, as they use the same state variable
%define UNISON4     0x03

%macro  SU_OSCILLAT 8
    db  %2
    db  %3
    db  %4
    db  %5
    db  %6
    db  %7
    db  %8
    USE_OSCILLAT
    %xdefine CMDS CMDS OSCILLAT_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_OSCILLAT
    %endif
    %if (%8) & SINE == SINE
        %define INCLUDE_SINE
    %endif
    %if (%8) & TRISAW == TRISAW
        %define INCLUDE_TRISAW
    %endif
    %if (%8) & PULSE == PULSE
        %define INCLUDE_PULSE
    %endif
    %if (%8) & GATE == GATE
        %define INCLUDE_GATE
    %endif
    %if (%8) & SAMPLE == SAMPLE
        %define INCLUDE_SAMPLES
    %endif
    %if (%8) & UNISON4 > 0
        %define INCLUDE_UNISONS
    %endif
%endmacro

struc   su_osc_ports
    .transpose  resd    1
    .detune     resd    1
    .phaseofs   resd    1
    .color      resd    1
    .shape      resd    1
    .gain       resd    1
    .params
endstruc

struc   su_osc_wrk
    .phase      resd    1
    .gatestate  equ     16   ; we put is late so only UNISON3 and UNISON4 are unusable with gate
    .size
endstruc

%define TRANSPOSE(val)  val
%define DETUNE(val)     val
%define PHASE(val)      val
%define GATESLOW(val)   val
%define GATESHIGH(val)  val
%define COLOR(val)      val
%define SAMPLENO(val)   val
%define SHAPE(val)      val
%define FLAGS(val)      val

;-------------------------------------------------------------------------------
;   Sample related defines
;-------------------------------------------------------------------------------

%macro SU_BEGIN_SAMPLE_OFFSETS 0
    SECT_DATA(susamoff)

    EXPORT MANGLE_DATA(su_sample_offsets)
%endmacro

%macro SAMPLE_OFFSET 3
    dd  %1
    dw  %2
    dw  %3
%endmacro

%define START(val)      val
%define LOOPSTART(val)  val
%define LOOPLENGTH(val) val

%define SU_END_SAMPLE_OFFSETS

struc   su_sample_offset  ; length conveniently 8, so easy to index
    .start      resd    1
    .loopstart  resw    1
    .looplength resw    1
    .size
endstruc

;-------------------------------------------------------------------------------
;   NOISE structs
;-------------------------------------------------------------------------------
%assign NOISE_ID -1
%macro USE_NOISE 0
    %if NOISE_ID == -1
        %assign NOISE_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_noise,0),
        %xdefine NUMPARAMS NUMPARAMS 2,
    %endif
%endmacro

%macro  SU_NOISE 3
    db  %2
    db  %3
    USE_NOISE
    %xdefine CMDS CMDS NOISE_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_NOISE
    %endif
%endmacro

struc   su_noise_ports
    .shape      resd    1
    .gain       resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   LOAD_VAL structs
;-------------------------------------------------------------------------------
%assign LOADVAL_ID -1
%macro USE_LOAD_VAL 0
    %if LOADVAL_ID == -1
        %assign LOADVAL_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_loadval,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro  SU_LOADVAL 2
    db  %2
    USE_LOAD_VAL
    %xdefine CMDS CMDS LOADVAL_ID+%1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_LOADVAL
    %endif
%endmacro

%define VALUE(val)  val

struc   su_load_val_ports
    .value      resd    1
    .params
endstruc

struc   su_load_val_wrk
    .size
endstruc

;-------------------------------------------------------------------------------
;   RECEIVE structs
;-------------------------------------------------------------------------------
%assign RECEIVE_ID -1
%macro USE_RECEIVE 0
    %if RECEIVE_ID == -1
        %assign RECEIVE_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_receive,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro  SU_RECEIVE 1
    USE_RECEIVE
    %xdefine CMDS CMDS RECEIVE_ID+%1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_RECEIVE
    %endif
%endmacro

struc su_receive_ports
    .params  ; receive does not actually have params, left and right only through send
    .left       resd    1
    .right      resd    1
endstruc
