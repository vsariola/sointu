cmake_minimum_required(VERSION 3.15)

# This policy is needed so that we can set the MSVC_RUNTIME to statically linked
# i.e. set_property(TARGET 4klang PROPERTY MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
cmake_policy(SET CMP0091 NEW) 

project(sointu
    VERSION 0.0.0
    DESCRIPTION "A modular synthesizer for 4k/8k/64k intros"
    LANGUAGES C CXX)

# Only do these if this is the main project, and not if it is included through add_subdirectory
if(CMAKE_PROJECT_NAME STREQUAL PROJECT_NAME)
    # Let's ensure -std=c++xx instead of -std=g++xx
    set(CMAKE_CXX_EXTENSIONS OFF)

    # Let's nicely support folders in IDE's
    set_property(GLOBAL PROPERTY USE_FOLDERS ON)

    # Testing only available if this is the main app
    # Note this needs to be done in the main CMakeLists
    # since it calls enable_testing, which must be in the
    # main CMakeLists.
    include(CTest)
endif()

enable_language(ASM_NASM)

# The normal NASM compile object does not include <DEFINES>
# By putting them there, we can pass the same compile definitions to C and ASM
set(CMAKE_ASM_NASM_COMPILE_OBJECT "<CMAKE_ASM_NASM_COMPILER> <INCLUDES> <DEFINES> <FLAGS> -f ${CMAKE_ASM_NASM_OBJECT_FORMAT} -o <OBJECT> <SOURCE>")

# Sointu as a header-only library 
set(HEADERLIB sointuinterface)
add_library(${HEADERLIB} INTERFACE)
target_include_directories(${HEADERLIB} INTERFACE ${PROJECT_SOURCE_DIR}/include)

# Sointu as static library
set(STATICLIB sointu)
add_library(${STATICLIB} render.asm)
set_target_properties(${STATICLIB} PROPERTIES LINKER_LANGUAGE C)
target_link_libraries(${STATICLIB} ${HEADERLIB})
target_compile_definitions(${STATICLIB} PUBLIC SU_USE_INTROSPECTION RUNTIME_TABLES)

# We should put examples here
# add_subdirectory(examples)

# Testing only available if this is the main app
# Emergency override 4KLANG_CMAKE_BUILD_TESTING provided as well
if((CMAKE_PROJECT_NAME STREQUAL PROJECT_NAME OR SOINTU_CMAKE_BUILD_TESTING) AND BUILD_TESTING)
    add_subdirectory(tests)
endif()