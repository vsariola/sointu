{{- if not .Song.Output16Bit }}
            (i64.store (global.get $outputBufPtr) (i64.load (i32.const 4128))) ;; load the sample from left & right channels as one 64bit int and store it in the address pointed by outputBufPtr
            (global.set $outputBufPtr (i32.add (global.get $outputBufPtr) (i32.const 8)))      ;; advance outputbufptr
{{- else }}
            (local.set $channel (i32.const 0))
            loop $channelLoop
                (i32.store16 (global.get $outputBufPtr) (i32.trunc_f32_s
                    (f32.mul
                        (call $clip
                            (f32.load offset=4128 (i32.mul (local.get $channel) (i32.const 4)))
                        )
                        (f32.const 32767)
                    )
                ))
                (global.set $outputBufPtr (i32.add (global.get $outputBufPtr) (i32.const 2)))
                (br_if $channelLoop (local.tee $channel (i32.eqz (local.get $channel))))
            end
{{- end }}
            (i64.store (i32.const 4128) (i64.const 0)) ;; clear the left and right ports