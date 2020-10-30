%ifdef SU_USE_INTROSPECTION

; Various compile time definitions exported
SECT_DATA(introscn)


%ifdef SU_USE_16BIT_OUTPUT
    EXPORT MANGLE_DATA(su_use_16bit_output)
        dd 1
%else
    EXPORT MANGLE_DATA(su_use_16bit_output)
        dd 0
%endif

%ifndef SU_DISABLE_PLAYER
    %ifdef MAX_SAMPLES
        EXPORT MANGLE_DATA(su_max_samples)
             dd MAX_SAMPLES
    %endif
%endif

; Arithmetic opcode ids
EXPORT MANGLE_DATA(su_add_id)
    dd ADD_ID
EXPORT MANGLE_DATA(su_addp_id)
    dd ADDP_ID
EXPORT MANGLE_DATA(su_pop_id)
    dd POP_ID
EXPORT MANGLE_DATA(su_loadnote_id)
    dd LOADNOTE_ID
EXPORT MANGLE_DATA(su_mul_id)

    dd MUL_ID
EXPORT MANGLE_DATA(su_mulp_id)
      dd MULP_ID
EXPORT MANGLE_DATA(su_push_id)
      dd PUSH_ID
EXPORT MANGLE_DATA(su_xch_id)
       dd XCH_ID

; Effect opcode ids
EXPORT MANGLE_DATA(su_distort_id)
   dd DISTORT_ID
EXPORT MANGLE_DATA(su_hold_id)
      dd HOLD_ID
EXPORT MANGLE_DATA(su_crush_id)
     dd CRUSH_ID
EXPORT MANGLE_DATA(su_gain_id)
      dd GAIN_ID
EXPORT MANGLE_DATA(su_invgain_id)
   dd INVGAIN_ID
EXPORT MANGLE_DATA(su_filter_id)
    dd FILTER_ID
EXPORT MANGLE_DATA(su_clip_id)
      dd CLIP_ID
EXPORT MANGLE_DATA(su_pan_id)
       dd PAN_ID
EXPORT MANGLE_DATA(su_delay_id)
     dd DELAY_ID
EXPORT MANGLE_DATA(su_compres_id)
   dd COMPRES_ID

; Flowcontrol opcode ids
EXPORT MANGLE_DATA(su_advance_id)
   dd SU_ADVANCE_ID
EXPORT MANGLE_DATA(su_speed_id)
     dd SPEED_ID

; Sink opcode ids
EXPORT MANGLE_DATA(su_out_id)
       dd OUT_ID
EXPORT MANGLE_DATA(su_outaux_id)
    dd OUTAUX_ID
EXPORT MANGLE_DATA(su_aux_id)
       dd AUX_ID
EXPORT MANGLE_DATA(su_send_id)
      dd SEND_ID

; Source opcode ids
EXPORT MANGLE_DATA(su_envelope_id)
  dd ENVELOPE_ID
EXPORT MANGLE_DATA(su_noise_id)
     dd NOISE_ID
EXPORT MANGLE_DATA(su_oscillat_id)
  dd OSCILLAT_ID
EXPORT MANGLE_DATA(su_loadval_id)
   dd LOADVAL_ID
EXPORT MANGLE_DATA(su_receive_id)
   dd RECEIVE_ID
EXPORT MANGLE_DATA(su_in_id)
    dd IN_ID

%endif ; SU_USE_INTROSPECTION
