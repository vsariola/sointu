%define WRK ebp             ; // alias for unit workspace
%define VAL esi             ; // alias for unit values (transformed/untransformed)
%define COM ebx             ; // alias for instrument opcodes

;===============================================================================
;   Uninitialized data: The one and only synth object
;===============================================================================
SECT_BSS(susynth)

su_synth_obj            resb    su_synth.size
su_transformed_values   resd    16

;===============================================================================
; The opcode table jump table. This is constructed to only include the opcodes
; that are used so that the jump table is as small as possible.
;===============================================================================
SECT_DATA(suoptabl)

su_synth_commands
                        dd      OPCODES

;===============================================================================
; The number of transformed parameters each opcode takes
;===============================================================================
SECT_DATA(suparcnt)

su_opcode_numparams
                        db      NUMPARAMS

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
    mov     COM, MANGLE_DATA(su_commands)           ; COM points to vm code
    mov     VAL, MANGLE_DATA(su_params)             ; VAL points to unit params
    ; su_unit.size will be added back before WRK is used
    mov     WRK, su_synth_obj + su_synth.voices + su_voice.workspace - su_unit.size
    push    COM                                     ; Stack: COM
    push    VAL                                     ; Stack: VAL COM
    push    WRK                                     ; Stack: WRK VAL COM
%if DELAY_ID > -1    
    mov     dword [MANGLE_DATA(su_delay_buffer_ofs)], MANGLE_DATA(su_delay_buffer) ; reset delaywrk to first delayline
%endif
    xor     ecx, ecx                                ; voice = 0
    push    ecx                                     ; Stack: voice WRK VAL COM
su_run_vm_loop:                                     ; loop until all voices done
    movzx   eax, byte [COM]                         ; eax = command byte
    inc     COM                                     ; move to next instruction
    add     WRK, su_unit.size                       ; move WRK to next unit
    push    eax
    shr     eax,1
    mov     al,byte [eax+su_opcode_numparams]
    push    eax
    call    su_transform_values
    mov     ecx, dword [esp+8]
    pop     eax
    shr     eax,1
    call    dword [eax*4+su_synth_commands]         ; call the function corresponding to the instruction
    cmp     dword [esp],MAX_VOICES                  ; if (voice < MAX_VOICES)
    jl      su_run_vm_loop                          ;   goto vm_loop
    add     esp, 16                                 ; Stack cleared
    ret

;-------------------------------------------------------------------------------
;   FloatRandomNumber function
;-------------------------------------------------------------------------------
;   Output:     st0     :   result
;-------------------------------------------------------------------------------
SECT_TEXT(surandom)

EXPORT MANGLE_FUNC(FloatRandomNumber,0)
    push    eax
    imul    eax,dword [MANGLE_DATA(RandSeed)],16007
    mov     dword [MANGLE_DATA(RandSeed)], eax
    fild    dword [MANGLE_DATA(RandSeed)]
    fidiv   dword [c_RandDiv]
    pop     eax
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
    push    ecx
    xor     ecx, ecx
    xor     eax, eax
    mov     edx, su_transformed_values
su_transform_values_loop:
    cmp     ecx, dword [esp+8]
    jge     su_transform_values_out
    lodsb
    push    eax
    fild    dword [esp]
    fmul    dword [c_i128]
    fadd    dword [WRK+su_unit.ports+ecx*4]
    fstp    dword [edx+ecx*4]
    mov     dword [WRK+su_unit.ports+ecx*4], 0
    pop     eax
    inc     ecx
    jmp     su_transform_values_loop
su_transform_values_out:
    pop     ecx
    ret     4

%macro TRANSFORM_VALUES 1
    push %1 %+ .params/4
    call su_transform_values
%endmacro

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
    fld     dword [edx+eax*4]   ; x, where x is the parameter in the range 0-1
    fimul   dword [c_24]        ; 24*x
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
%include "player.asm"
%include "introspection.asm"
