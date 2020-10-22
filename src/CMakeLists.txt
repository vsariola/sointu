set(LIB sointu)

set(SOURCES sointu.asm)

# Headers
include_directories(${PROJECT_SOURCE_DIR}/include)

# Library target
add_library(${LIB} ${SOURCES})
set_target_properties(${LIB} PROPERTIES LINKER_LANGUAGE C)
target_link_libraries(${LIB})
target_compile_definitions(${LIB} PUBLIC SU_USE_INTROSPECTION)
