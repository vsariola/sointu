add_executable(test_envelope test_envelope.asm test_renderer.c test_envelope_expected.raw)
add_test(test_envelope test_envelope)

add_executable(test_load test_load.asm test_renderer.c test_load_expected.raw)
add_test(test_load test_load)

add_executable(test_panning test_panning.asm test_renderer.c test_panning_expected.raw)
add_test(test_panning test_panning)

add_executable(test_multiple_instruments test_multiple_instruments.asm test_renderer.c test_multiple_instruments_expected.raw)
add_test(test_multiple_instruments test_multiple_instruments)

set_tests_properties(test_envelope PROPERTIES FIXTURES_SETUP ENVELOPE)
set_tests_properties(test_panning PROPERTIES FIXTURES_REQUIRED ENVELOPE)
set_tests_properties(test_multiple_instruments PROPERTIES FIXTURES_REQUIRED ENVELOPE)

file(GLOB RAW_FILES "${CMAKE_SOURCE_DIR}/tests/*.raw")
file(COPY ${RAW_FILES} DESTINATION ${CMAKE_CURRENT_BINARY_DIR})