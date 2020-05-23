%if BITS == 64
    %define WRK rbp ; alias for unit workspace
    %define VAL rsi ; alias for unit values (transformed/untransformed)
    %define COM rbx ; alias for instrument opcodes
    %define INP rdx ; alias for transformed inputs
    %define _AX rax ; push and offsets have to be r* on 64-bit and e* on 32-bit
    %define _BX rbx
    %define _CX rcx
    %define _DX rdx
    %define _SP rsp
    %define _SI rsi
    %define _DI rdi
    %define _BP rbp
    %define PTRSIZE 8
    %define PTRWORD qword
    %define RESPTR resq
    %define DPTR dq

    %macro apply 2
        mov r9, qword %2
        %1 [r9]
    %endmacro

    %macro apply 3
        mov r9, qword %2
        %1 [r9] %3
    %endmacro

    %macro apply 4
        mov r9, qword %2
        %1 [r9+%3] %4
    %endmacro

    %macro apply 5
        mov r9, qword %2
        lea r9, [r9+%3]
        %1 [r9+%4] %5
    %endmacro

    %macro  push_registers 1-*
        %rep  %0
            push    %1
            %rotate 1
        %endrep
    %endmacro

    %macro  pop_registers 1-*
        %rep %0
            %rotate -1
            pop     %1
        %endrep
    %endmacro

    %define PUSH_REG_SIZE(n) (n*8)
%else
    %define WRK ebp ; alias for unit workspace
    %define VAL esi ; alias for unit values (transformed/untransformed)
    %define COM ebx ; alias for instrument opcodes
    %define INP edx ; alias for transformed inputs
    %define _AX eax
    %define _BX ebx
    %define _CX ecx
    %define _DX edx
    %define _SP esp
    %define _SI esi
    %define _DI edi
    %define _BP ebp
    %define PTRSIZE 4
    %define PTRWORD dword
    %define RESPTR resd
    %define DPTR dd

    %macro apply 2
        %1 [%2]
    %endmacro

    %macro apply 3
        %1 [%2] %3
    %endmacro

    %macro apply 4
        %1 [%2+%3] %4
    %endmacro

    %macro apply 5
        %1 [%2+%3+%4] %5
    %endmacro

    %macro  push_registers 1-*
        pushad ; in 32-bit mode, this is the easiest way to store all the registers
    %endmacro

    %macro  pop_registers 1-*
        popad
    %endmacro

    %define PUSH_REG_SIZE(n) 32
%endif

struc su_stack ; the structure of stack _as the units see it_
    .retaddr    RESPTR  1
    .voiceno    RESPTR  1
    .wrk        RESPTR  1
    .val        RESPTR  1
    .com        RESPTR  1
%if DELAY_ID > -1
    .delaywrk   RESPTR  1
%endif
    .retaddrvm  RESPTR  1
    .rowtick    RESPTR  1    ; which tick within this row are we at
    .row        RESPTR  1    ; which total row of the song are we at
    .tick       RESPTR  1    ; which total tick of the song are we at
endstruc

;===============================================================================
;   Uninitialized data: The one and only synth object
;===============================================================================
SECT_BSS(susynth)

su_synth_obj            resb    su_synth.size

;===============================================================================
; The opcode table jump table. This is constructed to only include the opcodes
; that are used so that the jump table is as small as possible.
;===============================================================================
SECT_DATA(suoptabl)

su_synth_commands       DPTR    OPCODES

;===============================================================================
; The number of transformed parameters each opcode takes
;===============================================================================
SECT_DATA(suparcnt)

su_opcode_numparams     db      NUMPARAMS

;-------------------------------------------------------------------------------
;   Constants used by the common functions
;-------------------------------------------------------------------------------
SECT_DATA(suconst)

c_i128                  dd      0.0078125
c_RandDiv               dd      65536*32768
c_0_5                   dd      0.5
EXPORT MANGLE_DATA(RandSeed)
                        dd      1
c_24                    dd      24
c_i12                   dd      0x3DAAAAAA
EXPORT MANGLE_DATA(LFO_NORMALIZE)
                        dd      DEF_LFO_NORMALIZE

%ifdef INCLUDE_POLYPHONY
su_polyphony_bitmask    dd      POLYPHONY_BITMASK ; does the next voice reuse the current opcodes?
%endif

;-------------------------------------------------------------------------------
;   su_run_vm function: runs the entire virtual machine once, creating 1 sample
;-------------------------------------------------------------------------------
;   Input:      su_synth_obj.left   :   Set to 0 before calling
;               su_synth_obj.right  :   Set to 0 before calling
;   Output:     su_synth_obj.left   :   left sample
;               su_synth_obj.right  :   right sample
;   Dirty:      everything
;-------------------------------------------------------------------------------
SECT_TEXT(surunvm)

EXPORT MANGLE_FUNC(su_run_vm,0)
%if DELAY_ID > -1
    %if BITS == 64 ; TODO: find a way to do this with a macro
        mov     _AX,PTRWORD MANGLE_DATA(su_delay_buffer-su_delayline_wrk.filtstate)
        push    _AX                                 ; reset delaywrk to first delayline
    %else
        push    PTRWORD MANGLE_DATA(su_delay_buffer-su_delayline_wrk.filtstate)
    %endif
%endif
    mov     COM, PTRWORD MANGLE_DATA(su_commands)           ; COM points to vm code
    mov     VAL, PTRWORD MANGLE_DATA(su_params)             ; VAL points to unit params
    mov     WRK, PTRWORD su_synth_obj + su_synth.voices     ; WRK points to the first voice
    push    COM                                     ; Stack: COM
    push    VAL                                     ; Stack: VAL COM
    push    WRK                                     ; Stack: WRK VAL COM
    xor     ecx, ecx                                ; voice = 0
    push    _CX                                     ; Stack: voice WRK VAL COM
su_run_vm_loop:                                     ; loop until all voices done
    movzx   eax, byte [COM]                         ; eax = command byte
    inc     COM                                     ; move to next instruction
    add     WRK, su_unit.size                       ; move WRK to next unit
    push    _AX
    shr     eax,1
    apply {mov al,byte},su_opcode_numparams,_AX,{}
    push    _AX
    call    su_transform_values
    pop     _AX
    shr     eax,1
    apply call,su_synth_commands,_AX*PTRSIZE,{}     ; call the function corresponding to the instruction
    cmp     dword [_SP+su_stack.voiceno-PTRSIZE],MAX_VOICES ; if (voice < MAX_VOICES)
    jl      su_run_vm_loop                          ;   goto vm_loop
    add     _SP, su_stack.retaddrvm-PTRSIZE         ; Stack cleared
    ret

;-------------------------------------------------------------------------------
;   FloatRandomNumber function
;-------------------------------------------------------------------------------
;   Output:     st0     :   result
;-------------------------------------------------------------------------------
SECT_TEXT(surandom)

EXPORT MANGLE_FUNC(FloatRandomNumber,0)
    push    _AX
    apply {imul eax,},MANGLE_DATA(RandSeed),{,16007}
    apply mov,MANGLE_DATA(RandSeed),{, eax}
    apply fild dword,MANGLE_DATA(RandSeed)
    apply fidiv dword,c_RandDiv
    pop     _AX
    ret

;-------------------------------------------------------------------------------
;   su_transform_values function: transforms values and adds modulations
;-------------------------------------------------------------------------------
;   Input:      [esp]   :   number of bytes to transform
;               VAL     :   pointer to byte stream
;   Output:     eax     :   last transformed byte (zero extended)
;               edx     :   pointer to su_transformed_values, containing
;                           each byte transformed as x/128.0f+modulations
;               VAL     :   updated to point after the transformed bytes
;-------------------------------------------------------------------------------
SECT_TEXT(sutransf)

su_transform_values:
    xor     ecx, ecx
    xor     eax, eax
    mov     INP, [_SP+su_stack.wrk+2*PTRSIZE]
    add     INP, su_voice.inputs
su_transform_values_loop:
    cmp     ecx, dword [_SP+PTRSIZE]
    jnb     su_transform_values_out
    lodsb
    push    _AX
    fild    dword [_SP]
    apply fmul dword, c_i128
    fadd    dword [WRK+su_unit.ports+_CX*4]
    fstp    dword [INP+_CX*4]
    mov     dword [WRK+su_unit.ports+_CX*4], 0
    pop     _AX
    inc     ecx
    jmp     su_transform_values_loop
su_transform_values_out:
    ret     PTRSIZE

;-------------------------------------------------------------------------------
;   su_env_map function: computes 2^(-24*x) of the envelope parameter
;-------------------------------------------------------------------------------
;   Input:      eax     :   envelope parameter (0 = attac, 1 = decay...)
;               edx     :   pointer to su_transformed_values
;   Output:     st0     :   2^(-24*x), where x is the parameter in the range 0-1
;-------------------------------------------------------------------------------
SECT_TEXT(supower)

%if ENVELOPE_ID > -1 ; TODO: compressor also uses this, so should be compiled if either
su_env_map:
    fld     dword [INP+_AX*4]   ; x, where x is the parameter in the range 0-1
    apply   fimul dword,c_24          ; 24*x
    fchs                        ; -24*x
    ; flow into Power function, which outputs 2^(-24*x)
%endif

;-------------------------------------------------------------------------------
;   su_power function: computes 2^x
;-------------------------------------------------------------------------------
;   Input:      st0     :   x
;   Output:     st0     :   2^x
;-------------------------------------------------------------------------------
EXPORT MANGLE_FUNC(su_power,0)
    fld1          ; 1 x
    fld st1       ; x 1 x
    fprem         ; mod(x,1) 1 x
    f2xm1         ; 2^mod(x,1)-1 1 x
    faddp st1,st0 ; 2^mod(x,1) x
    fscale        ; 2^mod(x,1)*2^trunc(x) x
                  ; Equal to:
                  ; 2^x x
    fstp st1      ; 2^x
    ret

;-------------------------------------------------------------------------------
;   Include the rest of the code
;-------------------------------------------------------------------------------
%include "opcodes/arithmetic.asm"
%include "opcodes/flowcontrol.asm"
%include "opcodes/sources.asm"
%include "opcodes/sinks.asm"
; warning: at the moment effects has to be assembled after
; sources, as sources.asm defines SU_USE_WAVESHAPER
; if needed.
%include "opcodes/effects.asm"
%include "introspection.asm"
%include "player.asm"

%ifidn __OUTPUT_FORMAT__,win64
    %include "win64/gmdls_win64.asm"
%endif

%ifidn __OUTPUT_FORMAT__,win32
    %include "win32/gmdls_win32.asm"
%endif