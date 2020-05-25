;-------------------------------------------------------------------------------
;   OUT structs
;-------------------------------------------------------------------------------
%assign OUT_ID -1
%macro USE_OUT 0
    %if OUT_ID == -1
        %assign OUT_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_out,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro  SU_OUT 2
    db  %2
    USE_OUT
    %xdefine CMDS CMDS OUT_ID+%1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_OUT
    %endif
%endmacro

%define GAIN(val)       val

struc   su_out_ports
    .gain       resd    1
    .params
endstruc

;-------------------------------------------------------------------------------
;   OUTAUX structs
;-------------------------------------------------------------------------------
%assign OUTAUX_ID -1
%macro USE_OUTAUX 0
    %if OUTAUX_ID == -1
        %assign OUTAUX_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_outaux,0),
        %xdefine NUMPARAMS NUMPARAMS 2,
    %endif
%endmacro

%macro  SU_OUTAUX 3
    db  %2
    db  %3
    USE_OUTAUX
    %xdefine CMDS CMDS OUTAUX_ID+%1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_OUTAUX
    %endif
%endmacro

%define OUTGAIN(val)    val
%define AUXGAIN(val)    val

struc   su_outaux_ports
    .outgain    resd    1
    .auxgain    resd    1
endstruc

;-------------------------------------------------------------------------------
;   AUX defines
;-------------------------------------------------------------------------------
%assign AUX_ID -1
%macro USE_AUX 0
    %if AUX_ID == -1
        %assign AUX_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_aux,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro  SU_AUX 3
    db  %2
    db  %3
    USE_AUX
    %xdefine CMDS CMDS AUX_ID+%1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_AUX
    %endif
%endmacro

%define CHANNEL(val)    val

struc   su_aux_ports
    .gain       resd    1
endstruc

;-------------------------------------------------------------------------------
;   SEND structs
;-------------------------------------------------------------------------------
%assign SEND_ID -1
%macro USE_SEND 0
    %if SEND_ID == -1
        %assign SEND_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_send,0),
        %xdefine NUMPARAMS NUMPARAMS 1,
    %endif
%endmacro

%macro  SU_SEND 3
    db  %2
    dw  %3
    USE_SEND
    %xdefine CMDS CMDS SEND_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_SEND
    %endif
    %if (%3) & SEND_GLOBAL == SEND_GLOBAL
        %define INCLUDE_GLOBAL_SEND
    %endif
%endmacro

%define AMOUNT(val)     val
%define PORT(unit,unittype,port) ((unit+1)*su_unit.size + su_ %+ unittype %+ _ports. %+ port + su_unit.ports)/4
%define GLOBALPORT(voice,unit,unittype,port) SEND_GLOBAL + (su_synth.voices+voice*su_voice.size+su_voice.workspace+unit*su_unit.size + su_ %+ unittype %+ _ports. %+ port + su_unit.ports)/4
%define OUTPORT         0
%define SEND_POP        0x8000
%define SEND_GLOBAL     0x4000

struc   su_send_ports
    .amount     resd    1
    .params
endstruc
