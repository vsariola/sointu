{{- if .Opcode "pop"}}
;-------------------------------------------------------------------------------
;   POP opcode: remove (discard) the topmost signal from the stack
;-------------------------------------------------------------------------------
{{- if .Mono "pop" -}}
;   Mono:   a -> (empty)
{{- end}}
{{- if .Stereo "pop" -}}
;   Stereo: a b -> (empty)
{{- end}}
;-------------------------------------------------------------------------------
{{.Func "su_op_pop" "Opcode"}}
{{- if .StereoAndMono "pop"}}
    jnc     su_op_pop_mono
{{- end}}
{{- if .Stereo "pop"}}
    fstp    st0
{{- end}}
{{- if .StereoAndMono "pop"}}
su_op_pop_mono:
{{- end}}
    fstp    st0
    ret
{{end}}


{{- if .Opcode "add"}}
;-------------------------------------------------------------------------------
;   ADD opcode: add the two top most signals on the stack
;-------------------------------------------------------------------------------
{{- if .Mono "add"}}
;   Mono:   a b -> a+b b
{{- end}}
{{- if .Stereo "add" -}}
;   Stereo: a b c d -> a+c b+d c d
{{- end}}
;-------------------------------------------------------------------------------
{{.Func "su_op_add" "Opcode"}}
{{- if .StereoAndMono "add"}}
    jnc     su_op_add_mono
{{- end}}
{{- if .Stereo "add"}}
    fadd    st0, st2
    fxch
    fadd    st0, st3
    fxch
    ret
{{- end}}
{{- if .StereoAndMono "add"}}
su_op_add_mono:
{{- end}}
{{- if .Mono "add"}}
    fadd    st1
{{- end}}
{{- if .Mono "add"}}
    ret
    {{- end}}
{{end}}


{{- if .Opcode "addp"}}
;-------------------------------------------------------------------------------
;   ADDP opcode: add the two top most signals on the stack and pop
;-------------------------------------------------------------------------------
;   Mono:   a b -> a+b
;   Stereo: a b c d -> a+c b+d
;-------------------------------------------------------------------------------
{{.Func "su_op_addp" "Opcode"}}
{{- if .StereoAndMono "addp"}}
    jnc     su_op_addp_mono
{{- end}}
{{- if .Stereo "addp"}}
    faddp   st2, st0
    faddp   st2, st0
    ret
{{- end}}
{{- if .StereoAndMono "addp"}}
su_op_addp_mono:
{{- end}}
{{- if (.Mono "addp")}}
    faddp   st1, st0
    ret
{{- end}}
{{end}}


{{- if .Opcode "loadnote"}}
;-------------------------------------------------------------------------------
;   LOADNOTE opcode: load the current note, scaled to [-1,1]
;-------------------------------------------------------------------------------
{{if (.Mono "loadnote") -}}  ;   Mono:   (empty) -> n, where n is the note{{end}}
{{if (.Stereo "loadnote") -}};   Stereo: (empty) -> n n{{end}}
;-------------------------------------------------------------------------------
{{.Func "su_op_loadnote" "Opcode"}}
{{- if .StereoAndMono "loadnote"}}
    jnc     su_op_loadnote_mono
{{- end}}
{{- if .Stereo "loadnote"}}
    call    su_op_loadnote_mono
    su_op_loadnote_mono:
{{- end}}
    fild    dword [{{.INP}}-su_voice.inputs+su_voice.note]
    {{.Prepare (.Float 0.0078125)}}
    fmul    dword [{{.Use (.Float 0.0078125)}}]  ; s=n/128.0
    {{.Prepare (.Float 0.5)}}
    fsub    dword [{{.Use (.Float 0.5)}}]        ; s-.5
    fadd    st0, st0                        ; 2*s-1
    ret
{{end}}


{{- if .Opcode "mul"}}
;-------------------------------------------------------------------------------
;   MUL opcode: multiply the two top most signals on the stack
;-------------------------------------------------------------------------------
;   Mono:   a b -> a*b a
;   Stereo: a b c d -> a*c b*d c d
;-------------------------------------------------------------------------------
{{.Func "su_op_mul" "Opcode"}}
    jnc su_op_mul_mono
    fmul    st0, st2
    fxch
    fadd    st0, st3
    fxch
    ret
su_op_mul_mono:
    fmul    st1
    ret
{{end}}


{{- if .Opcode "mulp"}}
;-------------------------------------------------------------------------------
;   MULP opcode: multiply the two top most signals on the stack and pop
;-------------------------------------------------------------------------------
;   Mono:   a b -> a*b
;   Stereo: a b c d -> a*c b*d
;-------------------------------------------------------------------------------
{{.Func "su_op_mulp" "Opcode"}}
{{- if .StereoAndMono "mulp"}}
    jnc     su_op_mulp_mono
{{- end}}
{{- if .Stereo "mulp"}}
    fmulp   st2, st0
    fmulp   st2, st0
    ret
{{- end}}
{{- if .StereoAndMono "mulp"}}
su_op_mulp_mono:
{{- end}}
{{- if .Mono "mulp"}}
    fmulp   st1
    ret
{{- end}}
{{end}}


{{- if .Opcode "push"}}
;-------------------------------------------------------------------------------
;   PUSH opcode: push the topmost signal on the stack
;-------------------------------------------------------------------------------
;   Mono:   a -> a a
;   Stereo: a b -> a b a b
;-------------------------------------------------------------------------------
{{.Func "su_op_push" "Opcode"}}
{{- if .StereoAndMono "push"}}
    jnc     su_op_push_mono
{{- end}}
{{- if .Stereo "push"}}
    fld     st1
    fld     st1
    ret
{{- end}}
{{- if .StereoAndMono "push"}}
su_op_push_mono:
{{- end}}
{{- if .Mono "push"}}
    fld     st0
    ret
    {{- end}}
{{end}}


{{- if .Opcode "xch"}}
;-------------------------------------------------------------------------------
;   XCH opcode: exchange the signals on the stack
;-------------------------------------------------------------------------------
;   Mono:   a b -> b a
;   stereo: a b c d -> c d a b
;-------------------------------------------------------------------------------
{{.Func "su_op_xch" "Opcode"}}
{{- if .StereoAndMono "xch"}}
    jnc     su_op_xch_mono
{{- end}}
{{- if .Stereo "xch"}}
    fxch    st0, st2 ; c b a d
    fxch    st0, st1 ; b c a d
    fxch    st0, st3 ; d c a b
{{- end}}
{{- if .StereoAndMono "xch"}}
su_op_xch_mono:
{{- end}}
    fxch    st0, st1
    ret
{{end}}
