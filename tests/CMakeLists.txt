add_executable(test_env test_env.asm test_renderer.c test_env_expected.raw)
add_custom_command(
        TARGET test_env POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E copy
                ${CMAKE_SOURCE_DIR}/tests/test_env_expected.raw
                ${CMAKE_CURRENT_BINARY_DIR}/test_env_expected.raw)
add_test(test_env test_env)