set(LIB sointu)

set(SOURCES sointu.asm)

# Headers
include_directories(${PROJECT_SOURCE_DIR}/include)

# Library target
add_library(${LIB} ${SOURCES})
set_target_properties(${LIB} PROPERTIES LINKER_LANGUAGE C)
target_link_libraries(${LIB})
target_compile_definitions(${LIB} PUBLIC SU_USE_INTROSPECTION)

# Generate cgo wrapper
add_custom_command(TARGET ${LIB}
                   POST_BUILD
                   COMMAND ${CMAKE_COMMAND} -P ${PROJECT_SOURCE_DIR}/BindConfig.txt ${PROJECT_SOURCE_DIR} ${PROJECT_SOURCE_DIR}/include $<TARGET_FILE:sointu>
)
