;-------------------------------------------------------------------------------
;   OUT opcode: outputs and pops the signal
;-------------------------------------------------------------------------------
;   Mono: add ST0 to global left port
;   Stereo: also add ST1 to global right port
;-------------------------------------------------------------------------------
%if OUT_ID > -1

SECT_TEXT(suopout)

EXPORT MANGLE_FUNC(su_op_out,0) ; l r
    mov     _AX, PTRWORD su_synth_obj + su_synth.left
    %ifdef INCLUDE_STEREO_OUT
        jnc     su_op_out_mono
        call    su_op_out_mono
        add     _AX, 4
    su_op_out_mono:
    %endif
    fmul    dword [INP+su_out_ports.gain] ; g*l
    fadd    dword [_AX]                   ; g*l+o
    fstp    dword [_AX]                   ; o'=g*l+o
    ret

%endif ; SU_OUT_ID > -1

;-------------------------------------------------------------------------------
;   SEND opcode: adds the signal to a port
;-------------------------------------------------------------------------------
;   Mono: adds signal to a memory address, defined by a word in VAL stream
;   Stereo: also add right signal to the following address
;-------------------------------------------------------------------------------
%if SEND_ID > -1

SECT_TEXT(susend)

EXPORT MANGLE_FUNC(su_op_send,0)
    lodsw
    mov     _CX, [_SP + su_stack.wrk]
%ifdef INCLUDE_STEREO_SEND
    jnc     su_op_send_mono
    mov     _DI, _AX
    inc     _AX  ; send the right channel first
    fxch                        ; r l
    call    su_op_send_mono     ; (r) l
    mov     _AX, _DI            ; move back to original address
    test    _AX, SEND_POP       ; if r was not popped and is still in the stack
    jnz     su_op_send_mono
    fxch                        ; swap them back: l r
su_op_send_mono:
%endif
%ifdef INCLUDE_GLOBAL_SEND
    test    _AX, SEND_GLOBAL
    jz      su_op_send_skipglobal
    mov     _CX, PTRWORD su_synth_obj
su_op_send_skipglobal:
%endif
    test    _AX, SEND_POP       ; if the SEND_POP bit is not set
    jnz     su_op_send_skippush
    fld     st0                 ; duplicate the signal on stack: s s
su_op_send_skippush:            ; there is signal s, but maybe also another: s (s)
    fld     dword [INP+su_send_ports.amount]   ; a l (l)
    apply   fsub dword, c_0_5                        ; a-.5 l (l)
    fadd    st0                                ; g=2*a-1 l (l)
    and     _AX, 0x0000ffff - SEND_POP - SEND_GLOBAL ; eax = send address
    fmulp   st1, st0                           ; g*l (l)
    fadd    dword [_CX + _AX*4]     ; g*l+L (l),where L is the current value
    fstp    dword [_CX + _AX*4]     ; (l)
    ret

%endif ; SU_USE_SEND > -1
