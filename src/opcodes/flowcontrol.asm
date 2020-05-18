;-------------------------------------------------------------------------------
;   su_op_advance function: opcode to advance from one instrument to next
;-------------------------------------------------------------------------------
;   Stack:      voice wrkptr valptr comptr
;               voice   :   the number of voice we are currently processing
;               wrkptr  :   pointer to the first unit of current voice - su_unit.size
;               valptr  :   pointer to the first unit's value of current voice
;               comptr  :   pointer to the first command of current voice
;               COM     :   pointer to the command after current command
;   Output:     WRK     :   pointer to the next unit to be processed
;               VAL     :   pointer to the values of the next to be processed
;               COM     :   pointer to the next command to be executed
;
;   Checks if this was the last voice of current instrument. If so, moves to
;   next opcodes and updates the stack to reflect the instrument change.
;   If this instrument has more voices to process, rolls back the COM and VAL
;   pointers to where they were when this instrument started.
;-------------------------------------------------------------------------------
SECT_TEXT(suopadvn)

%ifdef INCLUDE_POLYPHONY

EXPORT MANGLE_FUNC(su_op_advance,0)     ; Stack: addr voice wrkptr valptr comptr
    mov     WRK, dword [esp+8]          ; WRK = wrkptr
    add     WRK, su_voice.size          ; move to next voice
    mov     dword [esp+8], WRK          ; update stack
    mov     ecx, dword [esp+4]          ; ecx = voice
    bt      dword [su_polyphony_bitmask],ecx ; if voice bit of su_polyphonism not set
    jnc     su_op_advance_next_instrument ; goto next_instrument
    mov     VAL, dword [esp+12]         ; rollback to where we were earlier
    mov     COM, dword [esp+16]
    jmp     short su_op_advance_finish
su_op_advance_next_instrument:
    mov     dword [esp+12], VAL         ; save current VAL as a checkpoint
    mov     dword [esp+16], COM         ; save current COM as a checkpoint
su_op_advance_finish:
    inc     dword [esp+4]
    ret

%else

EXPORT MANGLE_FUNC(su_op_advance,0)     ; Stack: addr voice wrkptr valptr comptr
    mov     WRK, dword [esp+8]          ; WRK = wrkptr
    add     WRK, su_voice.size          ; move to next voice
    mov     dword [esp+8], WRK          ; update stack
    inc     dword [esp+4]               ; voice++
    ret

%endif

;-------------------------------------------------------------------------------
;   SPEED tick
;-------------------------------------------------------------------------------
%if SPEED_ID > -1

SECT_TEXT(suspeed)

EXPORT MANGLE_FUNC(su_op_speed,0)
    fsub    dword [c_0_5]                ; s-.5
    fadd    st0, st0                     ; 2*s-1
    fmul    dword [c_bpmscale]           ; (2*s-1)*64/24, let's call this p from now on
    call    MANGLE_FUNC(su_power,0)      ; 2^p, this is how many ticks we should be taking
    fld1                                 ; 1 2^p
    fsubp   st1, st0                     ; 2^p-1, the player is advancing 1 tick by its own
    fadd    dword [WRK+su_speed_wrk.remainder] ; t+2^p-1, t is the remainder from previous rounds as ticks have to be rounded to 1
    push    eax
    fist    dword [esp]                  ; Main stack: k=int(t+2^p-1)
    fisub   dword [esp]                  ; t+2^p-1-k, the remainder
    pop     eax
    add     dword [esp+24], eax          ; add the whole ticks to song tick count, [esp+24] is the current tick in the row
    fstp    dword [WRK+su_speed_wrk.remainder] ; save the remainder for future
    ret

SECT_DATA(suconst)
    c_bpmscale      dd      2.666666666666 ; 64/24, 24 values will be double speed, so you can go from ~ 1/2.5 speed to 2.5x speed

%endif

;-------------------------------------------------------------------------------
;    Constants
;-------------------------------------------------------------------------------
%ifdef SU_USE_DLL_DC_FILTER
%ifndef C_DC_CONST
SECT_DATA(suconst)
c_dc_const              dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
%define C_DC_CONST
%endif

%endif
