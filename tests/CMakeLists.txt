add_executable(test_envelope test_envelope.asm test_renderer.c test_envelope_expected.raw)
add_test(test_envelope test_envelope)

add_executable(test_load test_load.asm test_renderer.c test_load_expected.raw)
add_test(test_load test_load)

file(GLOB RAW_FILES "${CMAKE_SOURCE_DIR}/tests/*.raw")
file(COPY ${RAW_FILES} DESTINATION ${CMAKE_CURRENT_BINARY_DIR})