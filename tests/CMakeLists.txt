add_executable(test_envelope test_envelope.asm test_renderer.c)
add_test(test_envelope test_envelope)

file(GLOB RAW_FILES "${CMAKE_SOURCE_DIR}/tests/*.raw")
file(COPY ${RAW_FILES} DESTINATION ${CMAKE_CURRENT_BINARY_DIR})