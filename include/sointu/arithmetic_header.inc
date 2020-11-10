;-------------------------------------------------------------------------------
;   ADDP related defines
;-------------------------------------------------------------------------------
%assign ADDP_ID -1
%macro USE_ADDP 0
    %if ADDP_ID == -1
        %assign ADDP_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_addp,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_ADDP 1
    USE_ADDP
    %xdefine CMDS CMDS ADDP_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_ADDP
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   ADD related defines
;-------------------------------------------------------------------------------
%assign ADD_ID -1
%macro USE_ADD 0
    %if ADD_ID == -1
        %assign ADD_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_add,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%assign ADD_ID -1
%macro SU_ADD 1
    USE_ADD
    %xdefine CMDS CMDS ADD_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_ADD
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   POP related defines
;-------------------------------------------------------------------------------
%assign POP_ID -1
%macro USE_POP 0
    %if POP_ID == -1
        %assign POP_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_pop,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_POP 1
    USE_POP
    %xdefine CMDS CMDS POP_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_POP
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   LOADNOTE related defines
;-------------------------------------------------------------------------------
%assign LOADNOTE_ID -1
%macro USE_LOADNOTE 0
    %if LOADNOTE_ID == -1
        %assign LOADNOTE_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_loadnote,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_LOADNOTE 1
    USE_LOADNOTE
    %xdefine CMDS CMDS LOADNOTE_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_LOADNOTE
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   MUL related defines
;-------------------------------------------------------------------------------
%assign MUL_ID -1
%macro USE_MUL 0
    %if MUL_ID == -1
        %assign MUL_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_mul,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_MUL 1
    USE_MUL
    %xdefine CMDS CMDS MUL_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_MUL
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   MULP related defines
;-------------------------------------------------------------------------------
%assign MULP_ID -1
%macro USE_MULP 0
    %if MULP_ID == -1
        %assign MULP_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_mulp,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_MULP 1
    USE_MULP
    %xdefine CMDS CMDS MULP_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_MULP
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   PUSH related defines
;-------------------------------------------------------------------------------
%assign PUSH_ID -1
%macro USE_PUSH 0
    %if PUSH_ID == -1
        %assign PUSH_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_push,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_PUSH 1
    USE_PUSH
    %xdefine CMDS CMDS PUSH_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_PUSH
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   XCH related defines
;-------------------------------------------------------------------------------
%assign XCH_ID -1
%macro USE_XCH 0
    %if XCH_ID == -1
        %assign XCH_ID CUR_ID
        %assign CUR_ID CUR_ID + 2
        %xdefine OPCODES OPCODES MANGLE_FUNC(su_op_xch,0),
        %xdefine NUMPARAMS NUMPARAMS 0,
    %endif
%endmacro

%macro SU_XCH 1
    USE_XCH
    %xdefine CMDS CMDS XCH_ID + %1,
    %if %1 == 1
        %define INCLUDE_STEREO_XCH
    %endif
%endmacro
