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
    inc     ecx                         ; voice++
    mov     dword [esp+4], ecx
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
;    Constants
;-------------------------------------------------------------------------------
%ifdef SU_USE_DLL_DC_FILTER
%ifndef C_DC_CONST
SECT_DATA(suconst)
c_dc_const              dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
%define C_DC_CONST
%endif

%endif