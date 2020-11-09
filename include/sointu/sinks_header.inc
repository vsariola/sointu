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

%macro  SU_SEND 6  ; global send (params: STEREO, AMOUNT, VOICE, UNIT, PORT, SENDPOP)
    db  %2
    %if (%3) > 0 
        dw  SEND_GLOBAL + (su_synthworkspace.voices+(%3-1)*su_voice.size+su_voice.workspace+%4*su_unit.size + su_unit.ports)/4 + %5 + (SENDPOPFLAG * %6)
        %define INCLUDE_GLOBAL_SEND
    %else
        dw  ((%4+1)*su_unit.size + su_unit.ports)/4 + %5 + (SENDPOPFLAG * %6)
    %endif
    USE_SEND
    %xdefine CMDS CMDS SEND_ID + %1,
    %if %1 == STEREO
        %define INCLUDE_STEREO_SEND
    %endif    
%endmacro

%define VOICE(val)      val
%define UNIT(val)       val
%define PORT(val)       val
%define AMOUNT(val)     val
%define OUTPORT         0
%define SENDPOP(val)    val
%define SENDPOPFLAG     0x8000
%define SEND_GLOBAL     0x4000

struc   su_send_ports
    .amount     resd    1
endstruc
