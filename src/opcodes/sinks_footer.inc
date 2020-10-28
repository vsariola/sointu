;-------------------------------------------------------------------------------
;   OUT opcode: outputs and pops the signal
;-------------------------------------------------------------------------------
;   Mono: add ST0 to main left port
;   Stereo: also add ST1 to main right port
;-------------------------------------------------------------------------------
%if OUT_ID > -1

SECT_TEXT(suopout)

EXPORT MANGLE_FUNC(su_op_out,0) ; l r
    mov     _AX, [_SP + su_stack.synth]
    %ifdef INCLUDE_STEREO_OUT
        jnc     su_op_out_mono
        call    su_op_out_mono
        add     _AX, 4
    su_op_out_mono:
    %endif
    fmul    dword [INP + su_out_ports.gain] ; g*l
    fadd    dword [_AX + su_synthworkspace.left]   ; g*l+o
    fstp    dword [_AX + su_synthworkspace.left]   ; o'=g*l+o
    ret

%endif ; SU_OUT_ID > -1

;-------------------------------------------------------------------------------
;   OUTAUX opcode: outputs to main and aux1 outputs and pops the signal
;-------------------------------------------------------------------------------
;   Mono: add outgain*ST0 to main left port and auxgain*ST0 to aux1 left
;   Stereo: also add outgain*ST1 to main right port and auxgain*ST1 to aux1 right
;-------------------------------------------------------------------------------
%if OUTAUX_ID > -1

SECT_TEXT(suoutaux)

EXPORT MANGLE_FUNC(su_op_outaux,0) ; l r
    mov     _AX, [_SP + su_stack.synth]
    %ifdef INCLUDE_STEREO_OUTAUX
        jnc     su_op_outaux_mono
        call    su_op_outaux_mono
        add     _AX, 4
    su_op_outaux_mono:
    %endif
    fld     st0                                     ; l l
    fmul    dword [INP + su_outaux_ports.outgain]   ; g*l
    fadd    dword [_AX + su_synthworkspace.left]             ; g*l+o
    fstp    dword [_AX + su_synthworkspace.left]             ; o'=g*l+o
    fmul    dword [INP + su_outaux_ports.auxgain]   ; h*l
    fadd    dword [_AX + su_synthworkspace.aux]              ; h*l+a
    fstp    dword [_AX + su_synthworkspace.aux]              ; a'=h*l+a
    ret

%endif ; SU_OUTAUX_ID > -1

;-------------------------------------------------------------------------------
;   AUX opcode: outputs the signal to aux (or main) port and pops the signal
;-------------------------------------------------------------------------------
;   Mono: add gain*ST0 to left port
;   Stereo: also add gain*ST1 to right port
;-------------------------------------------------------------------------------
%if AUX_ID > -1

SECT_TEXT(suopaux)

EXPORT MANGLE_FUNC(su_op_aux,0) ; l r
    lodsb
    mov     _DI, [_SP + su_stack.synth]
    %ifdef INCLUDE_STEREO_AUX
        jnc     su_op_aux_mono
        call    su_op_aux_mono
        add     _DI, 4
    su_op_aux_mono:
    %endif
    fmul    dword [INP + su_aux_ports.gain]     ; g*l
    fadd    dword [_DI + su_synthworkspace.left + _AX*4] ; g*l+o
    fstp    dword [_DI + su_synthworkspace.left + _AX*4] ; o'=g*l+o
    ret

%endif ; SU_AUX_ID > -1

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
    mov     _CX, [_SP + su_stack.synth]
su_op_send_skipglobal:
%endif
    test    _AX, SEND_POP       ; if the SEND_POP bit is not set
    jnz     su_op_send_skippush
    fld     st0                 ; duplicate the signal on stack: s s
su_op_send_skippush:            ; there is signal s, but maybe also another: s (s)
    fld     dword [INP+su_send_ports.amount]   ; a l (l)
 do fsub    dword [,c_0_5,]                    ; a-.5 l (l)
    fadd    st0                                ; g=2*a-1 l (l)
    and     _AX, 0x0000ffff - SEND_POP - SEND_GLOBAL ; eax = send address
    fmulp   st1, st0                           ; g*l (l)
    fadd    dword [_CX + _AX*4]     ; g*l+L (l),where L is the current value
    fstp    dword [_CX + _AX*4]     ; (l)
    ret

%endif ; SU_USE_SEND > -1
