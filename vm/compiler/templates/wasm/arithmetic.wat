{{- if .HasOp "pop"}}
;;-------------------------------------------------------------------------------
;;   POP opcode: remove (discard) the topmost signal from the stack
;;-------------------------------------------------------------------------------
{{- if .Mono "pop" -}}
;;   Mono:   a -> (empty)
{{- end}}
{{- if .Stereo "pop" -}}
;;   Stereo: a b -> (empty)
{{- end}}
;;-------------------------------------------------------------------------------
(func $su_op_pop (param $stereo i32)
{{- if .Stereo "pop"}}
    (if (local.get $stereo) (then
        (drop (call $pop))
    ))
{{- end}}
    (drop (call $pop))
)
{{end}}


{{- if .HasOp "add"}}
;;-------------------------------------------------------------------------------
;;   ADD opcode: add the two top most signals on the stack
;;-------------------------------------------------------------------------------
{{- if .Mono "add"}}
;;   Mono:   a b -> a+b b
{{- end}}
{{- if .Stereo "add" -}}
;;   Stereo: a b c d -> a+c b+d c d
{{- end}}
;;-------------------------------------------------------------------------------
(func $su_op_add (param $stereo i32)
{{- if .StereoAndMono "add"}}
    (if (local.get $stereo) (then
{{- end}}
{{- if .Stereo "add"}}
        call $pop  ;; F: b c d         P: a
        call $pop  ;; F: c d           P: b a
        call $peek2;; F: c d           P: d b a
        f32.add    ;; F: c d           P: b+d a
        call $push ;; F: b+d c d       P: a
        call $peek2;; F: b+d c d       P: c a
        f32.add    ;; F: b+d c d       P: a+c
        call $push ;; F: a+c b+d c d   P:
{{- end}}
{{- if .StereoAndMono "add"}}
    )(else
{{- end}}
{{- if .Mono "add"}}
        (call $push (f32.add (call $pop) (call $peek)))
{{- end}}
{{- if .StereoAndMono "add"}}
    ))
{{- end}}
)
{{end}}


{{- if .HasOp "addp"}}
;;-------------------------------------------------------------------------------
;;   ADDP opcode: add the two top most signals on the stack and pop
;;-------------------------------------------------------------------------------
;;   Mono:   a b -> a+b
;;   Stereo: a b c d -> a+c b+d
;;-------------------------------------------------------------------------------
(func $su_op_addp (param $stereo i32)
{{- if .StereoAndMono "addp"}}
    (if (local.get $stereo) (then
{{- end}}
{{- if .Stereo "addp"}}
        call $pop  ;; a
        call $pop  ;; b a
        call $swap ;; a b
        call $pop  ;; c a b
        f32.add    ;; c+a b
        call $swap ;; b c+a
        call $pop  ;; d b c+a
        f32.add    ;; d+b c+a
        call $push ;; c+a
        call $push
{{- end}}
{{- if .StereoAndMono "addp"}}
    )(else
{{- end}}
{{- if .Mono "addp"}}
        (call $push (f32.add (call $pop) (call $pop)))
{{- end}}
{{- if .StereoAndMono "addp"}}
    ))
{{- end}}
)
{{end}}


{{- if .HasOp "loadnote"}}
;;-------------------------------------------------------------------------------
;;   LOADNOTE opcode: load the current note, scaled to [-1,1]
;;-------------------------------------------------------------------------------
(func $su_op_loadnote (param $stereo i32)
{{- if .Stereo "loadnote"}}
    (if (local.get $stereo) (then
        (call $su_op_loadnote (i32.const 0))
    ))
{{- end}}
    (f32.convert_i32_u (i32.load (global.get $voice)))
    (f32.mul (f32.const 0.015625))
    (f32.sub (f32.const 1))
    (call $push)
)
{{end}}

{{- if .HasOp "mul"}}
;;-------------------------------------------------------------------------------
;;   MUL opcode: multiply the two top most signals on the stack
;;-------------------------------------------------------------------------------
;;   Mono:   a b -> a*b a
;;   Stereo: a b c d -> a*c b*d c d
;;-------------------------------------------------------------------------------
(func $su_op_mul (param $stereo i32)
{{- if .StereoAndMono "mul"}}
    (if (local.get $stereo) (then
{{- end}}
{{- if .Stereo "mul"}}
        call $pop  ;; F: b c d         P: a
        call $pop  ;; F: c d           P: b a
        call $peek2;; F: c d           P: d b a
        f32.mul    ;; F: c d           P: b*d a
        call $push ;; F: b*d c d       P: a
        call $peek2;; F: b*d c d       P: c a
        f32.mul    ;; F: b*d c d       P: a*c
        call $push ;; F: a*c b*d c d   P:
{{- end}}
{{- if .StereoAndMono "mul"}}
    )(else
{{- end}}
{{- if .Mono "mul"}}
        (call $push (f32.mul (call $pop) (call $peek)))
{{- end}}
{{- if .StereoAndMono "mul"}}
    ))
{{- end}}
)
{{end}}


{{- if .HasOp "mulp"}}
;;-------------------------------------------------------------------------------
;;   MULP opcode: multiply the two top most signals on the stack and pop
;;-------------------------------------------------------------------------------
;;   Mono:   a b -> a*b
;;   Stereo: a b c d -> a*c b*d
;;-------------------------------------------------------------------------------
(func $su_op_mulp (param $stereo i32)
{{- if .StereoAndMono "mulp"}}
    (if (local.get $stereo) (then
{{- end}}
{{- if .Stereo "mulp"}}
        call $pop  ;; a
        call $pop  ;; b a
        call $swap ;; a b
        call $pop  ;; c a b
        f32.mul    ;; c*a b
        call $swap ;; b c*a
        call $pop  ;; d b c*a
        f32.mul    ;; d*b c*a
        call $push ;; c*a
        call $push
{{- end}}
{{- if .StereoAndMono "mulp"}}
    )(else
{{- end}}
{{- if .Mono "mulp"}}
        (call $push (f32.mul (call $pop) (call $pop)))
{{- end}}
{{- if .StereoAndMono "mulp"}}
    ))
{{- end}}
)
{{end}}


{{- if .HasOp "push"}}
;;-------------------------------------------------------------------------------
;;   PUSH opcode: push the topmost signal on the stack
;;-------------------------------------------------------------------------------
;;   Mono:   a -> a a
;;   Stereo: a b -> a b a b
;;-------------------------------------------------------------------------------
(func $su_op_push (param $stereo i32)
{{- if .Stereo "push"}}
    (if (local.get $stereo) (then
        (call $push (call $peek))
    ))
{{- end}}
    (call $push (call $peek))
)
{{end}}


{{- if or (.HasOp "xch") (.Stereo "delay")}}
;;-------------------------------------------------------------------------------
;;   XCH opcode: exchange the signals on the stack
;;-------------------------------------------------------------------------------
;;   Mono:   a b -> b a
;;   stereo: a b c d -> c d a b
;;-------------------------------------------------------------------------------
(func $su_op_xch (param $stereo i32)
    call $pop
    call $pop
{{- if .StereoAndMono "xch"}}
    (if (local.get $stereo) (then
{{- end}}
{{- if .Stereo "xch"}}
        call $pop  ;; F: d       P: c b a
        call $swap ;; F: d       P: b c a
        call $pop  ;; F:         P: d b c a
        call $swap ;; F:         P: b d c a
        call $push ;; F: b       P: d c a
        call $push ;; F: d b     P: c a
        call $swap ;; F: d b     P: a c
        call $pop  ;; F: b       P: d a c
        call $swap ;; F: b       P: a d c
        call $push ;; F: a b     P: d c
{{- end}}
{{- if .StereoAndMono "xch"}}
    )(else
{{- end}}
{{- if or (.Mono "xch") (.Stereo "delay")}}
        call $swap
{{- end}}
{{- if .StereoAndMono "xch"}}
    ))
{{- end}}
    call $push
    call $push
)
{{end}}
