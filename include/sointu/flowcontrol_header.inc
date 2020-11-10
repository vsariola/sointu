;-------------------------------------------------------------------------------
;   SPEED related defines
;-------------------------------------------------------------------------------
%assign SPEED_ID -1
%macro USE_SPEED 0
    %if SPEED_ID == -1
        %assign SPEED_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_speed,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_SPEED 0
    USE_SPEED
    %xdefine CMDS CMDS SPEED_ID,
%endmacro

struc su_speed_wrk
    .remainder      resd    1
endstruc