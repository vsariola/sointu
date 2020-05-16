SECT_TEXT(suarithm)

;-------------------------------------------------------------------------------
;   op_pop function: a -> (empty)
;            stereo: a b -> (empty) 
;-------------------------------------------------------------------------------
%if POP_ID > -1

EXPORT MANGLE_FUNC(su_op_pop,0)
%ifdef INCLUDE_STEREO_POP
    jnc su_op_pop_mono
    fstp    st0
su_op_pop_mono:
%endif
    fstp    st0
    ret

%endif

;-------------------------------------------------------------------------------
;   op_add function: a b -> a+b b
;            stereo: a b c d -> a+c b+d c d 
;-------------------------------------------------------------------------------
%if ADD_ID > -1

EXPORT MANGLE_FUNC(su_op_add,0)
%ifdef INCLUDE_STEREO_ADD
    jnc su_op_add_mono
    fadd    st0, st2
    fxch
    fadd    st0, st3
    fxch
    ret
su_op_pop_mono:
%endif
    fadd    st1
    ret

%endif

;-------------------------------------------------------------------------------
;   op_addp function: a b -> a+b
;             stereo: a b c d -> a+c b+d
;-------------------------------------------------------------------------------
%if ADDP_ID > -1

EXPORT MANGLE_FUNC(su_op_addp,0)
%ifdef INCLUDE_STEREO_ADDP
    jnc su_op_addp_mono
    faddp   st2, st0
    faddp   st2, st0
    ret
su_op_addp_mono:
%endif
    faddp   st1, st0
    ret

%endif

;-------------------------------------------------------------------------------
;   op_loadnote function: (empty) -> n
;                 stereo: (empty) -> n n
;   ecx should point to the workspace (slightly offset)
;-------------------------------------------------------------------------------
%if LOADNOTE_ID > -1

EXPORT MANGLE_FUNC(su_op_loadnote,0)
%ifdef INCLUDE_STEREO_LOADNOTE
    jnc     su_op_loadnote_mono
    call    su_op_loadnote_mono
su_op_loadnote_mono:
%endif
    fild    dword [ecx+su_unit.size-su_voice.workspace+su_voice.note]
    fmul    dword [c_i128]
    ret

%endif

;-------------------------------------------------------------------------------
;   op_mul function: a b -> a*b a
;            stereo: a b c d -> a*c b*d c d
;-------------------------------------------------------------------------------
%if MUL_ID > -1

EXPORT MANGLE_FUNC(su_op_mul,0)
%ifdef INCLUDE_STEREO_MUL
    jnc su_op_mul_mono
    fmul    st0, st2
    fxch
    fadd    st0, st3
    fxch
    ret
su_op_mul_mono:
%endif
    fmul    st1
    ret

%endif

;-------------------------------------------------------------------------------
;   op_mulp function: a b -> a*b
;             stereo: a b c d -> a*c b*d
;-------------------------------------------------------------------------------
%if MULP_ID > -1

EXPORT MANGLE_FUNC(su_op_mulp,0)
%ifdef INCLUDE_STEREO_MULP
    jnc     su_op_mulp_mono
    fmulp   st2, st0
    fmulp   st2, st0
    ret
su_op_mulp_mono:
%endif
    fmulp   st1
    ret

%endif

;-------------------------------------------------------------------------------
;   op_push function: a -> a a
;             stereo: a b -> a b a b
;-------------------------------------------------------------------------------
%if PUSH_ID > -1

EXPORT MANGLE_FUNC(su_op_push,0)
%ifdef INCLUDE_STEREO_PUSH
    jnc     su_op_push_mono
    fld     st1
    fld     st1
    ret
su_op_push_mono:
%endif
    fld     st0
    ret

%endif

;-------------------------------------------------------------------------------
;   op_xch function: a b -> b a
;            stereo: a b c d -> c d a b
;-------------------------------------------------------------------------------
%if XCH_ID > -1

EXPORT MANGLE_FUNC(su_op_xch,0)
%ifdef INCLUDE_STEREO_XCH
    jnc     su_op_xch_mono
    fxch    st0, st2 ; c b a d
    fxch    st0, st1 ; b c a d
    fxch    st0, st2 ; d c a b
su_op_xch_mono:
%endif
    fxch    st0, st1
    ret

%endif