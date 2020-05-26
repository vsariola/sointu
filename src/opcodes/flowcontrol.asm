;-------------------------------------------------------------------------------
;   ADVANCE opcode: advances from one voice to next
;-------------------------------------------------------------------------------
;   Checks if this was the last voice of current instrument. If so, moves to
;   next opcodes and updates the stack to reflect the instrument change.
;   If this instrument has more voices to process, rolls back the COM and VAL
;   pointers to where they were when this instrument started.
;
;   There is no stereo version.
;-------------------------------------------------------------------------------
SECT_TEXT(suopadvn)

EXPORT MANGLE_FUNC(su_op_advance,0)
%ifdef INCLUDE_POLYPHONY
    mov     WRK, [_SP+su_stack.wrk]         ; WRK points to start of current voice
    add     WRK, su_voice.size              ; move to next voice
    mov     [_SP+su_stack.wrk], WRK         ; update the pointer in the stack to point to the new voice
    mov     ecx, [_SP+su_stack.voiceno]     ; ecx = how many voices remain to process
    dec     ecx                             ; decrement number of voices to process
    bt      dword [_SP+su_stack.polyphony], ecx ; if voice bit of su_polyphonism not set
    jnc     su_op_advance_next_instrument   ; goto next_instrument
    mov     VAL, PTRWORD [_SP+su_stack.val] ; if it was set, then repeat the opcodes for the current voice
    mov     COM, PTRWORD [_SP+su_stack.com]
su_op_advance_next_instrument:
    mov     PTRWORD [_SP+su_stack.val], VAL ; save current VAL as a checkpoint
    mov     PTRWORD [_SP+su_stack.com], COM ; save current COM as a checkpoint
su_op_advance_finish:
    mov     [_SP+su_stack.voiceno], ecx
    ret
%else
    mov     WRK, PTRWORD [_SP+su_stack.wrk] ; WRK = wrkptr
    add     WRK, su_voice.size              ; move to next voice
    mov     PTRWORD [_SP+su_stack.wrk], WRK ; update stack
    dec     PTRWORD [_SP+su_stack.voiceno]  ; voices--
    ret
%endif

;-------------------------------------------------------------------------------
;   SPEED opcode: modulate the speed (bpm) of the song based on ST0
;-------------------------------------------------------------------------------
;   Mono: adds or subtracts the ticks, a value of 0.5 is neutral & will7
;   result in no speed change.
;   There is no STEREO version.
;-------------------------------------------------------------------------------
%if SPEED_ID > -1

SECT_TEXT(suspeed)

EXPORT MANGLE_FUNC(su_op_speed,0)
 do fmul    dword [,c_bpmscale,]         ; (2*s-1)*64/24, let's call this p from now on
    call    MANGLE_FUNC(su_power,0)      ; 2^p, this is how many ticks we should be taking
    fld1                                 ; 1 2^p
    fsubp   st1, st0                     ; 2^p-1, the player is advancing 1 tick by its own
    fadd    dword [WRK+su_speed_wrk.remainder] ; t+2^p-1, t is the remainder from previous rounds as ticks have to be rounded to 1
    push    _AX
    fist    dword [_SP]                  ; Main stack: k=int(t+2^p-1)
    fisub   dword [_SP]                  ; t+2^p-1-k, the remainder
    pop     _AX
    add     dword [_SP+su_stack.rowtick], eax          ; add the whole ticks to row tick count
    fstp    dword [WRK+su_speed_wrk.remainder] ; save the remainder for future
    ret

%define USE_C_BPMSCALE

%endif
