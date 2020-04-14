add_executable(test_envelope test_envelope.asm test_renderer.c test_envelope_expected.raw)
add_test(test_envelope test_envelope)

add_executable(test_load test_load.asm test_renderer.c test_load_expected.raw)
add_test(test_load test_load)

add_executable(test_panning test_panning.asm test_renderer.c test_panning_expected.raw)
add_test(test_panning test_panning)

add_executable(test_multiple_instruments test_multiple_instruments.asm test_renderer.c test_multiple_instruments_expected.raw)
add_test(test_multiple_instruments test_multiple_instruments)

add_executable(test_fop_pop test_fop_pop.asm test_renderer.c test_fop_pop_expected.raw)
add_test(test_fop_pop test_fop_pop)

add_executable(test_fop_addp test_fop_addp.asm test_renderer.c test_fop_addp_expected.raw)
add_test(test_fop_addp test_fop_addp)

add_executable(test_fop_mulp test_fop_mulp.asm test_renderer.c test_fop_mulp_expected.raw)
add_test(test_fop_mulp test_fop_mulp)

add_executable(test_fop_push test_fop_push.asm test_renderer.c test_fop_push_expected.raw)
add_test(test_fop_push test_fop_push)

add_executable(test_fop_xch test_fop_xch.asm test_renderer.c test_fop_xch_expected.raw)
add_test(test_fop_xch test_fop_xch)

add_executable(test_fop_add test_fop_add.asm test_renderer.c test_fop_add_expected.raw)
add_test(test_fop_add test_fop_add)

add_executable(test_fop_mul test_fop_mul.asm test_renderer.c test_fop_mul_expected.raw)
add_test(test_fop_mul test_fop_mul)

add_executable(test_fop_addp2 test_fop_addp2.asm test_renderer.c test_fop_addp2_expected.raw)
add_test(test_fop_addp2 test_fop_addp2)

add_executable(test_fop_mulp2 test_fop_mulp2.asm test_renderer.c test_fop_mulp2_expected.raw)
add_test(test_fop_mulp2 test_fop_mulp2)

add_executable(test_fop_loadnote test_fop_loadnote.asm test_renderer.c test_fop_loadnote_expected.raw)
add_test(test_fop_loadnote test_fop_loadnote)

set_tests_properties(test_envelope PROPERTIES FIXTURES_SETUP ENVELOPE)
set_tests_properties(test_panning PROPERTIES FIXTURES_REQUIRED ENVELOPE)
set_tests_properties(test_multiple_instruments PROPERTIES FIXTURES_REQUIRED ENVELOPE)

set_tests_properties(test_load PROPERTIES FIXTURES_SETUP LOAD)
set_tests_properties(test_fop_pop PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_addp PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_mulp PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_push PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_xch PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_add PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_mul PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_addp2 PROPERTIES FIXTURES_REQUIRED LOAD)
set_tests_properties(test_fop_mulp2 PROPERTIES FIXTURES_REQUIRED LOAD)

set_tests_properties(test_fop_pop PROPERTIES FIXTURES_SETUP FOP_POP)
set_tests_properties(test_fop_push PROPERTIES FIXTURES_REQUIRED FOP_POP)

file(GLOB RAW_FILES "${CMAKE_SOURCE_DIR}/tests/*.raw")
file(COPY ${RAW_FILES} DESTINATION ${CMAKE_CURRENT_BINARY_DIR})