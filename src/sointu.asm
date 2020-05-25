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

    %macro do 2
        mov r9, qword %2
        %1 r9
    %endmacro

    %macro do 3
        mov r9, qword %2
        %1 r9 %3
    %endmacro

    %macro do 4
        mov r9, qword %2
        %1 r9+%3 %4
    %endmacro

    %macro do 5
        mov r9, qword %2
        lea r9, [r9+%3]
        %1 r9+%4 %5
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

    %macro do 2
        %1 %2
    %endmacro

    %macro do 3
        %1 %2 %3
    %endmacro

    %macro do 4
        %1 %2+%3 %4
    %endmacro

    %macro do 5
        %1 %2+%3+%4 %5
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
%if BITS == 32              ; we dump everything with pushad, so this is unused in 32-bit
                RESPTR  1
%endif
    .val        RESPTR  1
    .wrk        RESPTR  1
%if BITS == 32              ; we dump everything with pushad, so this is unused in 32-bit
                RESPTR  1
%endif
    .com        RESPTR  1
    .synth      RESPTR  1
    .delaywrk   RESPTR  1
%if BITS == 32              ; we dump everything with pushad, so this is unused in 32-bit
                RESPTR  1
%endif
    .retaddrvm  RESPTR  1
    .voiceno    RESPTR  1
%ifdef INCLUDE_POLYPHONY
    .polyphony  RESPTR  1
%endif
    .output_sound
    .rowtick    RESPTR  1    ; which tick within this row are we at
    .update_voices
    .row        RESPTR  1    ; which total row of the song are we at
    .tick       RESPTR  1    ; which total tick of the song are we at
    .randseed   RESPTR  1
%ifdef INCLUDE_MULTIVOICE_TRACKS
    .voicetrack RESPTR  1
%endif
    .render_epilogue
%if BITS == 32
                RESPTR  8   ; registers
    .retaddr_pl RESPTR  1
%elifidn __OUTPUT_FORMAT__,win64
                RESPTR  4   ; registers
%else
                RESPTR  2   ; registers
%endif
    .bufferptr  RESPTR  1
    .size
endstruc

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
c_24                    dd      24
c_i12                   dd      0x3DAAAAAA
EXPORT MANGLE_DATA(LFO_NORMALIZE)
                        dd      DEF_LFO_NORMALIZE

;-------------------------------------------------------------------------------
;   su_run_vm function: runs the entire virtual machine once, creating 1 sample
;-------------------------------------------------------------------------------
;   Input:      su_synth_obj.left   :   Set to 0 before calling
;               su_synth_obj.right  :   Set to 0 before calling
;               _CX                 :   Pointer to delay workspace (if needed)
;               _DX                 :   Pointer to synth object
;               COM                 :   Pointer to command stream
;               VAL                 :   Pointer to value stream
;               WRK                 :   Pointer to the last workspace processed
;               _DI                 :   Number of voices to process
;   Output:     su_synth_obj.left   :   left sample
;               su_synth_obj.right  :   right sample
;   Dirty:      everything
;-------------------------------------------------------------------------------
SECT_TEXT(surunvm)

EXPORT MANGLE_FUNC(su_run_vm,0)
    push_registers _CX, _DX, COM, WRK, VAL          ; save everything to stack
su_run_vm_loop:                                     ; loop until all voices done
    movzx   edi, byte [COM]                         ; edi = command byte
    inc     COM                                     ; move to next instruction
    add     WRK, su_unit.size                       ; move WRK to next unit
    shr     edi, 1                                  ; shift out the LSB bit = stereo bit
    mov     INP, [_SP+su_stack.wrk-PTRSIZE]         ; reset INP to point to the inputs part of voice
    add     INP, su_voice.inputs
    xor     ecx, ecx                                ; counter = 0
    xor     eax, eax                                ; clear out high bits of eax, as lodsb only sets al
su_transform_values_loop:
 do{cmp     cl, byte [},su_opcode_numparams,_DI,]   ; compare the counter to the value in the param count table
    je      su_transform_values_out
    lodsb                                           ; load the byte value from VAL stream
    push    _AX                                     ; push it to memory so FPU can read it
    fild    dword [_SP]                             ; load the value to FPU stack
 do fmul    dword [,c_i128,]                        ; divide it by 128 (0 => 0, 128 => 1.0)
    fadd    dword [WRK+su_unit.ports+_CX*4]         ; add the modulations in the current workspace
    fstp    dword [INP+_CX*4]                       ; store the modulated value in the inputs section of voice
    xor     eax, eax
    mov     dword [WRK+su_unit.ports+_CX*4], eax    ; clear out the modulation ports
    pop     _AX
    inc     ecx
    jmp     su_transform_values_loop
su_transform_values_out:
    bt      dword [COM-1],0                         ; LSB of COM = stereo bit => carry
 do call    [,su_synth_commands,_DI*PTRSIZE,]       ; call the function corresponding to the instruction
    cmp     dword [_SP+su_stack.voiceno-PTRSIZE],0  ; do we have more voices to process?
    jne     su_run_vm_loop                          ;   if there's more voices to process, goto vm_loop
    pop_registers _CX, _DX, COM, WRK, VAL           ; pop everything from stack
    ret

;-------------------------------------------------------------------------------
;   su_nonlinear_map function: returns 2^(-24*x) of parameter number _AX
;-------------------------------------------------------------------------------
;   Input:      _AX     :   parameter number (e.g. for envelope: 0 = attac, 1 = decay...)
;               INP     :   pointer to transformed values
;   Output:     st0     :   2^(-24*x), where x is the parameter in the range 0-1
;-------------------------------------------------------------------------------
SECT_TEXT(supower)

%if ENVELOPE_ID > -1 || COMPRES_ID > -1
su_nonlinear_map:
    fld     dword [INP+_AX*4]   ; x, where x is the parameter in the range 0-1
 do fimul   dword [,c_24,]      ; 24*x
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
