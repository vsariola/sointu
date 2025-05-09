function(regression_test testname)

    if(ARGV5)
        set(source ${ARGV5})
        add_executable(${testname} ${source} test_renderer.c)
    else()
        if(ARGV3)
            set(source ${ARGV3}.yml)
        else()
            set(source ${testname}.yml)
        endif()

        set(asmfile ${testname}.asm)
        set (headerfile ${CMAKE_CURRENT_BINARY_DIR}/${testname}.h)

        add_custom_command(
            OUTPUT ${asmfile}
            COMMAND ${compilecmd} ${ARGV4} -arch=${arch} -o ${CMAKE_CURRENT_BINARY_DIR}/${asmfile} ${CMAKE_CURRENT_SOURCE_DIR}/${source}
            DEPENDS ${source} ${x86templates} ${compilecmd}
        )

        add_executable(${testname} test_renderer.c ${asmfile})
        target_compile_definitions(${testname} PUBLIC TEST_HEADER=<${testname}.h>)

        if (NODE AND WAT2WASM AND NOT ${testname} MATCHES "sample" AND NOT ${testname} MATCHES "sync")
            set(wasmfile ${CMAKE_CURRENT_BINARY_DIR}/${testname}.wasm)
            set(watfile ${CMAKE_CURRENT_BINARY_DIR}/${testname}.wat)
            set(wasmtarget wasm_${testname})
            add_custom_command(
                OUTPUT ${wasmfile}
                COMMAND ${compilecmd} ${ARGV4} -arch=wasm -o ${watfile} ${CMAKE_CURRENT_SOURCE_DIR}/${source} && ${WAT2WASM} -o ${wasmfile} ${watfile}
                DEPENDS ${source} ${wasmtemplates} ${compilecmd}
            )

            add_custom_target(${wasmtarget} ALL
                SOURCES "${source}" "${wasmtemplates}"
                DEPENDS ${wasmfile}
            )

            add_test(${wasmtarget} ${NODE} ${CMAKE_CURRENT_SOURCE_DIR}/wasm_test_renderer.es6 ${wasmfile} ${CMAKE_CURRENT_SOURCE_DIR}/expected_output/${testname}.raw)
        endif()
    endif()

    if (${testname} MATCHES "sync")
        add_test(${testname} ${testname} ${CMAKE_CURRENT_SOURCE_DIR}/expected_output/${testname}.raw ${CMAKE_CURRENT_SOURCE_DIR}/expected_output/${testname}_syncbuf.raw)
    else()
        add_test(${testname} ${testname} ${CMAKE_CURRENT_SOURCE_DIR}/expected_output/${testname}.raw)
    endif()
    target_link_libraries(${testname} ${HEADERLIB})

    target_include_directories(${testname} PUBLIC ${CMAKE_CURRENT_BINARY_DIR})
    target_compile_definitions(${testname} PUBLIC TEST_NAME="${testname}")

    if (ARGV1)
        message("${testname} requires ${ARGV1}")
        set_tests_properties(${testname} PROPERTIES FIXTURES_REQUIRED "${ARGV1}")
    endif()

    if (ARGV2)
        message("${testname} setups ${ARGV2}")
        set_tests_properties(${testname} PROPERTIES FIXTURES_SETUP "${ARGV2}")
    endif()

endfunction(regression_test)

regression_test(test_envelope "" ENVELOPE)
regression_test(test_envelope_stereo ENVELOPE)
regression_test(test_out ENVELOPE)
regression_test(test_loadval "" LOADVAL)
regression_test(test_loadval_stereo LOADVAL LOADVAL_STEREO)
regression_test(test_gain LOADVAL GAIN)
regression_test(test_gain_stereo GAIN)
regression_test(test_invgain LOADVAL INVGAIN)
regression_test(test_invgain_stereo INVGAIN)
regression_test(test_dbgain LOADVAL DBGAIN)
regression_test(test_dbgain_stereo DBGAIN)
regression_test(test_send LOADVAL SEND)
regression_test(test_send_stereo SEND)
regression_test(test_send_global SEND SEND_GLOBAL)
regression_test(test_send_global_stereo SEND_GLOBAL)
regression_test(test_receive SEND RECEIVE)
regression_test(test_receive_stereo RECEIVE)
regression_test(test_in LOADVAL IN)
regression_test(test_in_stereo IN)
regression_test(test_outaux IN OUTAUX)
regression_test(test_outaux_stereo OUTAUX)
regression_test(test_aux LOADVAL AUX)
regression_test(test_aux_stereo AUX)
regression_test(test_panning ENVELOPE PANNING)
regression_test(test_panning_stereo PANNING)
regression_test(test_multiple_instruments ENVELOPE)
regression_test(test_pop LOADVAL POP)
regression_test(test_pop_stereo POP)
regression_test(test_addp LOADVAL)
regression_test(test_addp_stereo LOADVAL)
regression_test(test_mulp LOADVAL FOP_MULP)
regression_test(test_mulp_stereo LOADVAL FOP_MULP2)
regression_test(test_push "LOADVAL;POP" FOP_PUSH)
regression_test(test_push_stereo PUSH)
regression_test(test_xch LOADVAL)
regression_test(test_xch_stereo LOADVAL)
regression_test(test_add LOADVAL)
regression_test(test_add_stereo LOADVAL)
regression_test(test_mul LOADVAL)
regression_test(test_mul_stereo LOADVAL)
regression_test(test_loadnote)
regression_test(test_loadnote_stereo)
regression_test(test_noise ENVELOPE NOISE)
regression_test(test_noise_stereo NOISE)
regression_test(test_oscillat_sine ENVELOPE VCO_SINE)
regression_test(test_oscillat_trisaw ENVELOPE)
regression_test(test_oscillat_pulse ENVELOPE VCO_PULSE)
regression_test(test_oscillat_gate ENVELOPE)
regression_test(test_oscillat_stereo ENVELOPE)
if(WIN32) # The samples are currently only GMDLs based, and thus require Windows.
    regression_test(test_oscillat_sample ENVELOPE)
    regression_test(test_oscillat_sample_stereo ENVELOPE)
endif()
regression_test(test_oscillat_unison ENVELOPE)
regression_test(test_oscillat_unison_phase ENVELOPE)
regression_test(test_oscillat_unison_stereo ENVELOPE)
regression_test(test_oscillat_lfo "ENVELOPE;VCO_SINE;VCO_PULSE;FOP_MULP2")
regression_test(test_oscillat_transposemod "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_detunemod "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_phasemod "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_colormod "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_shapemod "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_gainmod OSCGAINMOD "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_gainmod_stereo "" OSCGAINMOD)
regression_test(test_oscillat_frequencymod "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")
regression_test(test_oscillat_frequencymod_stereo "VCO_SINE;ENVELOPE;FOP_MULP;FOP_PUSH;SEND")

regression_test(test_distort ENVELOPE)
regression_test(test_distort_mod "VCO_SINE;ENVELOPE;SEND")
regression_test(test_distort_stereo ENVELOPE)

regression_test(test_hold ENVELOPE HOLD)
regression_test(test_hold_mod "VCO_SINE;ENVELOPE;SEND")
regression_test(test_hold_stereo HOLD)

regression_test(test_clip "VCO_SINE;ENVELOPE;FOP_MULP;INVGAIN" CLIP)
regression_test(test_clip_stereo CLIP)

regression_test(test_crush "VCO_SINE;ENVELOPE;FOP_MULP;INVGAIN" CRUSH)
regression_test(test_crush_stereo CRUSH)

regression_test(test_compressor "" COMPRESSOR)
regression_test(test_compressor_stereo COMPRESSOR)

regression_test(test_filter_band "VCO_SINE;ENVELOPE;FOP_MULP")
regression_test(test_filter_low "VCO_SINE;ENVELOPE;FOP_MULP")
regression_test(test_filter_high "VCO_SINE;ENVELOPE;FOP_MULP")
regression_test(test_filter_peak "VCO_SINE;ENVELOPE;FOP_MULP")
regression_test(test_filter_stereo "VCO_SINE;ENVELOPE;FOP_MULP")
regression_test(test_filter_freqmod "VCO_SINE;ENVELOPE;FOP_MULP;SEND")
regression_test(test_filter_resmod "VCO_SINE;ENVELOPE;FOP_MULP;SEND")

regression_test(test_delay "ENVELOPE;FOP_MULP;PANNING;VCO_SINE")
regression_test(test_delay_stereo "ENVELOPE;FOP_MULP;PANNING;VCO_SINE")
regression_test(test_delay_notetracking "ENVELOPE;FOP_MULP;PANNING;NOISE")
regression_test(test_delay_notetracking_modulation "ENVELOPE;FOP_MULP;PANNING;NOISE")
regression_test(test_delay_reverb "ENVELOPE;FOP_MULP;PANNING;VCO_SINE")
regression_test(test_delay_feedbackmod "ENVELOPE;FOP_MULP;PANNING;VCO_SINE;SEND")
regression_test(test_delay_pregainmod "ENVELOPE;FOP_MULP;PANNING;VCO_SINE;SEND")
regression_test(test_delay_dampmod "ENVELOPE;FOP_MULP;PANNING;VCO_SINE;SEND")
regression_test(test_delay_drymod "ENVELOPE;FOP_MULP;PANNING;VCO_SINE;SEND")
regression_test(test_delay_flanger "ENVELOPE;FOP_MULP;PANNING;VCO_SINE;SEND")

regression_test(test_envelope_mod "VCO_SINE;ENVELOPE;SEND")
regression_test(test_envelope_16bit ENVELOPE "" test_envelope "-i")

regression_test(test_polyphony "ENVELOPE;VCO_SINE" POLYPHONY)
regression_test(test_polyphony_init POLYPHONY)
regression_test(test_chords "ENVELOPE;VCO_SINE")
regression_test(test_speed "ENVELOPE;VCO_SINE")
regression_test(test_sync "ENVELOPE" "" "" "-r")

regression_test(test_render_samples ENVELOPE "" "" "" test_render_samples.c)
target_link_libraries(test_render_samples ${STATICLIB})
target_compile_definitions(test_render_samples PUBLIC TEST_HEADER="test_render_samples.h")

add_executable(test_render_samples_api test_render_samples_api.c)
target_link_libraries(test_render_samples_api ${STATICLIB})
add_test(test_render_samples_api test_render_samples_api)

add_executable(test_oscillator_crash test_oscillator_crash.c)
target_link_libraries(test_oscillator_crash ${STATICLIB})
add_test(test_oscillator_crash test_oscillator_crash)

