{{- if .HasOp "out"}}
;-------------------------------------------------------------------------------
;   OUT opcode: outputs and pops the signal
;-------------------------------------------------------------------------------
{{- if .Mono "out"}}
;   Mono: add ST0 to main left port, then pop
{{- end}}
{{- if .Stereo "out"}}
;   Stereo: add ST0 to left out and ST1 to right out, then pop
{{- end}}
;-------------------------------------------------------------------------------
{{.Func "su_op_out" "Opcode"}}   ; l r
    mov     {{.AX}}, [{{.Stack "Synth"}}] ; AX points to the synth object
{{- if .StereoAndMono "out" }}
    jnc     su_op_out_mono
{{- end }}
{{- if .Stereo "out" }}
    call    su_op_out_mono
    add     {{.AX}}, 4 ; shift from left to right channel
su_op_out_mono:
{{- end}}
    fmul    dword [{{.Input "out" "gain"}}] ; multiply by gain
    fadd    dword [{{.AX}} + su_synthworkspace.left]   ; add current value of the output
    fstp    dword [{{.AX}} + su_synthworkspace.left]   ; store the new value of the output
    ret
{{end}}


{{- if .HasOp "outaux"}}
;-------------------------------------------------------------------------------
;   OUTAUX opcode: outputs to main and aux1 outputs and pops the signal
;-------------------------------------------------------------------------------
;   Mono: add outgain*ST0 to main left port and auxgain*ST0 to aux1 left
;   Stereo: also add outgain*ST1 to main right port and auxgain*ST1 to aux1 right
;-------------------------------------------------------------------------------
{{.Func "su_op_outaux" "Opcode"}} ; l r
    mov     {{.AX}}, [{{.Stack "Synth"}}]
{{- if .StereoAndMono "outaux" }}
    jnc     su_op_outaux_mono
{{- end}}
{{- if .Stereo "outaux" }}
    call    su_op_outaux_mono
    add     {{.AX}}, 4
su_op_outaux_mono:
{{- end}}
    fld     st0                                     ; l l
    fmul    dword [{{.Input "outaux" "outgain"}}]   ; g*l
    fadd    dword [{{.AX}} + su_synthworkspace.left]             ; g*l+o
    fstp    dword [{{.AX}} + su_synthworkspace.left]             ; o'=g*l+o
    fmul    dword [{{.Input "outaux" "auxgain"}}]   ; h*l
    fadd    dword [{{.AX}} + su_synthworkspace.aux]              ; h*l+a
    fstp    dword [{{.AX}} + su_synthworkspace.aux]              ; a'=h*l+a
    ret
{{end}}


{{- if .HasOp "aux"}}
;-------------------------------------------------------------------------------
;   AUX opcode: outputs the signal to aux (or main) port and pops the signal
;-------------------------------------------------------------------------------
;   Mono: add gain*ST0 to left port
;   Stereo: also add gain*ST1 to right port
;-------------------------------------------------------------------------------
{{.Func "su_op_aux" "Opcode"}} ; l r
    lodsb
    mov     {{.DI}}, [{{.Stack "Synth"}}]
{{- if .StereoAndMono "aux" }}
    jnc     su_op_aux_mono
{{- end}}
{{- if .Stereo "aux" }}
    call    su_op_aux_mono
    add     {{.DI}}, 4
su_op_aux_mono:
{{- end}}
    fmul    dword [{{.Input "aux" "gain"}}]     ; g*l
    fadd    dword [{{.DI}} + su_synthworkspace.left + {{.AX}}*4] ; g*l+o
    fstp    dword [{{.DI}} + su_synthworkspace.left + {{.AX}}*4] ; o'=g*l+o
    ret
{{end}}


{{- if .HasOp "send"}}
;-------------------------------------------------------------------------------
;   SEND opcode: adds the signal to a port
;-------------------------------------------------------------------------------
;   Mono: adds signal to a memory address, defined by a word in VAL stream
;   Stereo: also add right signal to the following address
;-------------------------------------------------------------------------------
{{.Func "su_op_send" "Opcode"}}
    lodsw
    mov     {{.CX}}, [{{.Stack "Voice"}}]  ; load pointer to voice
{{- if .SupportsParamValueOtherThan "send" "voice" 0}}
    pushf   ; uh ugly: we save the flags just for the stereo carry bit. Doing the .CX loading later crashed the synth for stereo sends as loading the synth address from stack was f'd up by the "call su_op_send_mono"
    test    {{.AX}}, 0x8000
    jz      su_op_send_skipglobal
    mov     {{.CX}}, [{{.Stack "Synth"}} + {{.PTRSIZE}}]
su_op_send_skipglobal:
    popf
{{- end}}
{{- if .StereoAndMono "send"}}
    jnc     su_op_send_mono
{{- end}}
{{- if .Stereo "send"}}
    mov     {{.DI}}, {{.AX}}
    inc     {{.AX}}  ; send the right channel first
    fxch                        ; r l
    call    su_op_send_mono     ; (r) l
    mov     {{.AX}}, {{.DI}}            ; move back to original address
    test    {{.AX}}, 0x8    ; if r was not popped and is still in the stack
    jnz     su_op_send_mono
    fxch                        ; swap them back: l r
su_op_send_mono:
{{- end}}
    test    {{.AX}}, 0x8        ; if the SEND_POP bit is not set
    jnz     su_op_send_skippush
    fld     st0                 ; duplicate the signal on stack: s s
su_op_send_skippush:            ; there is signal s, but maybe also another: s (s)
    fld     dword [{{.Input "send" "amount"}}]   ; a l (l)
{{- .Float 0.5 | .Prepare | indent 4}}
    fsub    dword [{{.Float 0.5 | .Use}}]                    ; a-.5 l (l)
    fadd    st0                                ; g=2*a-1 l (l)
    and     ah, 0x7f ; eax = send address, clear the global bit
    or      al, 0x8 ; set the POP bit always, at the same time shifting to ports instead of wrk
    fmulp   st1, st0                           ; g*l (l)
    fadd    dword [{{.CX}} + {{.AX}}*4]     ; g*l+L (l),where L is the current value
    fstp    dword [{{.CX}} + {{.AX}}*4]     ; (l)
    ret
{{end}}
