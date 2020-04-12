cmake_minimum_required(VERSION 3.14)

# This policy is needed so that we can set the MSVC_RUNTIME to statically linked
# i.e. set_property(TARGET 4klang PROPERTY MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
cmake_policy(SET CMP0091 NEW) 

project(4klang
    VERSION 3.3.2
    DESCRIPTION "A modular synthesizer for 4k/8k intros"
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

# The compiled VSTi is here
add_subdirectory(src)

# We should put examples here
# add_subdirectory(examples)
