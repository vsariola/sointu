{{template "structs.asm" .}}

struc su_synth
    .synth_wrk  resb    su_synthworkspace.size
    .delay_wrks resb    su_delayline_wrk.size * 128
    .delaytimes resw    768
    .sampleoffs resb    su_sample_offset.size * 256
    .randseed   resd    1
    .globaltime resd    1
    .opcodes    resb    32 * 64
    .operands   resb    32 * 64 * 8
    .polyphony  resd    1
    .numvoices  resd    1
endstruc

{{.ExportFunc "su_render" "SynthStateParam" "BufferPtrParam" "SamplesParam" "TimeParam"}}
    {{- if .Amd64}}
    {{- if eq .OS "windows"}}
    {{- .PushRegs "rdi" "NonVolatileRDI" "rsi" "NonVolatileRSI" "rbx" "NonVolatileRBX"  "rbp" "NonVolatileRBP"  | indent 4}}
    mov     rsi, r8 ; rsi = &samples
    mov     rbx, r9 ; rbx = &time
    {{- else}} ; SystemV amd64 ABI, linux mac or hopefully something similar
    {{- .PushRegs "rbx" "NonVolatileRBX"  "rbp" "NonVolatileRBP"  | indent 4}}
    mov     rbx, rcx ; rbx points to time
    xchg    rsi, rdx ; rdx points to buffer, rsi points to samples
    mov     rcx, rdi ; rcx = &Synthstate
    {{- end}}
    {{- else}}
    {{- .PushRegs | indent 4 }} ; push registers
    mov     ecx, [{{.Stack "SynthStateParam"}}] ; ecx = &synthState
    mov     edx, [{{.Stack "BufferPtrParam"}}]  ; edx = &buffer
    mov     esi, [{{.Stack "SamplesParam"}}]  ; esi = &samples
    mov     ebx, [{{.Stack "TimeParam"}}]  ; ebx = &time
    {{- end}}
    {{.SaveFPUState | indent 4}}       ; save the FPU state to stack & reset the FPU
    {{.Push .SI "Samples"}}
    {{.Push .BX "Time"}}
    xor     eax, eax    ; samplenumber starts at 0
    {{.Push .AX "BufSample"}}
    mov     esi, [{{.SI}}]  ; zero extend dereferenced pointer
    {{.Push .SI "BufSize"}}
    {{.Push .DX "BufPtr"}}
    {{.Push .CX "SynthState"}}
    lea     {{.AX}}, [{{.CX}} + su_synth.sampleoffs]
    {{.Push .AX "SampleTable"}}
    lea     {{.AX}}, [{{.CX}} + su_synth.delaytimes]
    {{.Push .AX "DelayTable"}}
    mov     eax, [{{.CX}} + su_synth.randseed]
    {{.Push .AX "RandSeed"}}
    mov     eax, [{{.CX}} + su_synth.globaltime]
    {{.Push .AX "GlobalTick"}}
    mov     ebx, dword [{{.BX}}]           ; zero extend dereferenced pointer
    {{.Push .BX "RowLength"}}             ; the nominal rowlength should be time_in
    xor     eax, eax                   ; rowtick starts at 0
su_render_samples_loop:
        push    {{.DI}}
        fnstsw  [{{.SP}}]                         ; store the FPU status flag to stack top
        pop     {{.DI}}                           ; {{.DI}} = FPU status flag
        and     {{.DI}}, 0011100001000101b        ; mask TOP pointer, stack error, zero divide and in{{.VAL}}id operation
        test    {{.DI}},{{.DI}}                       ; all the aforementioned bits should be 0!
        jne     su_render_samples_time_finish ; otherwise, we exit due to error
        cmp     eax, [{{.Stack "RowLength"}}]                    ; if rowtick >= maxtime
        jge     su_render_samples_time_finish ;   goto finish
        mov     ecx, [{{.Stack "BufSize"}}]        ; ecx = buffer length in samples
        cmp     [{{.Stack "BufSample"}}], ecx        ; if samples >= maxsamples
        jge     su_render_samples_time_finish ;   goto finish
        inc     eax                           ; time++
        inc     dword [{{.Stack "BufSample"}}]       ; samples++
        mov     {{.CX}}, [{{.Stack "SynthState"}}]
        {{.Push .AX "Sample"}}
        mov     eax, [{{.CX}} + su_synth.polyphony]
        {{.Push .AX "PolyphonyBitmask"}}
        mov     eax, [{{.CX}} + su_synth.numvoices]
        {{.Push .AX "VoicesRemain"}}
        lea     {{.DX}}, [{{.CX}}+ su_synth.synth_wrk]
        lea     {{.COM}}, [{{.CX}}+ su_synth.opcodes]
        lea     {{.VAL}}, [{{.CX}}+ su_synth.operands]
        lea     {{.WRK}}, [{{.DX}} + su_synthworkspace.voices]
        lea     {{.CX}}, [{{.CX}}+ su_synth.delay_wrks - su_delayline_wrk.filtstate]
        {{.Call "su_run_vm"}}
        {{.Pop .AX}}
        {{.Pop .AX}}
        mov     {{.DI}}, [{{.Stack "BufPtr"}}] ; edi containts buffer ptr
        mov     {{.CX}}, [{{.Stack "SynthState"}}]
        lea     {{.SI}}, [{{.CX}} + su_synth.synth_wrk + su_synthworkspace.left]
        movsd   ; copy left channel to output buffer
        movsd   ; copy right channel to output buffer
        mov     [{{.Stack "BufPtr"}}], {{.DI}} ; save back the updated ptr
        lea     {{.DI}}, [{{.SI}}-8]
        xor     eax, eax
        stosd   ; clear left channel so the VM is ready to write them again
        stosd   ; clear right channel so the VM is ready to write them again
        {{.Pop .AX}}
        inc     dword [{{.Stack "GlobalTick"}}] ; increment global time, used by delays
        jmp     su_render_samples_loop
su_render_samples_time_finish:
    {{.Pop .CX}}
    {{.Pop .BX}}
    {{.Pop .DX}}
    {{.Pop .CX}}
    {{.Pop .CX}}
    {{.Pop .CX}}
    mov     [{{.CX}} + su_synth.randseed], edx
    mov     [{{.CX}} + su_synth.globaltime], ebx
    {{.Pop .BX}}
    {{.Pop .BX}}
    {{.Pop .DX}}
    {{.Pop .BX}}
    {{.Pop .SI}}
    mov     dword [{{.SI}}], edx    ; *samples = samples rendered
    mov     dword [{{.BX}}], eax    ; *time = time ticks rendered
    mov     {{.AX}},{{.DI}}             ; {{.DI}} was the masked FPU status flag, {{.AX}} is return {{.VAL}}ue
    {{.LoadFPUState | indent 4}}       ; load the FPU state from stack
    {{- if .Amd64}}
    {{- if eq .OS "windows"}}
    {{- .PopRegs "rdi" "rsi" "rbx" "rbp" | indent 4}}
    {{- else}} ; SystemV amd64 ABI, linux mac or hopefully something similar
    {{- .PopRegs "rbx" "rbp" | indent 4}}
    {{- end}}
    ret
    {{- else}}
    mov     [{{.Stack "eax"}}],eax ; we want to return eax, but popad pops everything, so put eax to stack for popad to pop
    {{- .PopRegs | indent 4 }} ; popad
    ret     16
    {{- end}}


{{template "patch.asm" .}}

;-------------------------------------------------------------------------------
;    Constants
;-------------------------------------------------------------------------------
{{.SectData "constants"}}
{{.Constants}}
