cmake_minimum_required(VERSION 3.10)

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

IF(APPLE)
    set(CMAKE_EXE_LINKER_FLAGS "${CMAKE_EXE_LINKER_FLAGS} -Wl,-no_pie")
    # https://stackoverflow.com/questions/69803659/what-is-the-proper-way-to-build-for-macos-x86-64-using-cmake-on-apple-m1-arm
    set(CMAKE_OSX_ARCHITECTURES "x86_64" CACHE INTERNAL "" FORCE)
endif()

find_program(GO NAMES go)
if(NOT GO)
  message(FATAL_ERROR "go not found. Get it from: https://golang.org")
else()
  message("go found at: ${GO}")
endif()

find_program(NODE NAMES node)
if(NOT NODE)
  message( WARNING "node not found, cannot run WebAssembly tests. Get it from: https://nodejs.org/" )
else()
  message("node found at: ${NODE}")
endif()

find_program(WAT2WASM NAMES wat2wasm)
if(NOT WAT2WASM )
  message( WARNING "wat2wasm not found, cannot build wasm tests. Get it from: https://github.com/WebAssembly/wabt)" )
else()
  message("wat2wasm found at: ${WAT2WASM}")
endif()

enable_language(ASM_NASM)

# The normal NASM compile object does not include <DEFINES>
# By putting them there, we can pass the same compile definitions to C and ASM
set(CMAKE_ASM_NASM_COMPILE_OBJECT "<CMAKE_ASM_NASM_COMPILER> <INCLUDES> <DEFINES> <FLAGS> -f ${CMAKE_ASM_NASM_OBJECT_FORMAT} -o <OBJECT> <SOURCE>")

if(WIN32)
    set(compilecmd ${CMAKE_CURRENT_BINARY_DIR}/sointu-compile.exe)
else()
    set(compilecmd ${CMAKE_CURRENT_BINARY_DIR}/sointu-compile)
endif()

# the tests include the entire ASM but we still want to rebuild when they change
file(GLOB x86templates "${PROJECT_SOURCE_DIR}/vm/compiler/templates/amd64-386/*.asm")
file(GLOB wasmtemplates "${PROJECT_SOURCE_DIR}/vm/compiler/templates/wasm/*.wat")
file(GLOB sointusrc "${PROJECT_SOURCE_DIR}/*.go")
file(GLOB compilersrc "${PROJECT_SOURCE_DIR}/compiler/*.go")
file(GLOB compilecmdsrc "${PROJECT_SOURCE_DIR}/cmd/sointu-compile/*.go")

if(DEFINED CMAKE_C_SIZEOF_DATA_PTR AND CMAKE_C_SIZEOF_DATA_PTR EQUAL 8)
    set(arch "amd64")
elseif(DEFINED CMAKE_CXX_SIZEOF_DATA_PTR AND CMAKE_CXX_SIZEOF_DATA_PTR EQUAL 8)
    set(arch "amd64")
else()
    set(arch "386")
endif()

# Sointu as static library
set(STATICLIB sointu)
set(sointuasm sointu.asm)


# Build sointu-cli only once because go run has everytime quite a bit of delay when
# starting
add_custom_command(
    OUTPUT
        "${compilecmd}"
    COMMAND
        ${GO} build -o "${compilecmd}" ${PROJECT_SOURCE_DIR}/cmd/sointu-compile/main.go
    DEPENDS ${x86templates} ${wasmtemplates} ${sointusrc} ${compilersrc} ${compilecmdsrc}
)

add_custom_target(
    sointu-compiler
    DEPENDS ${compilecmd}
)

add_custom_command(
    OUTPUT ${sointuasm}
    COMMAND ${compilecmd} -arch=${arch} -a -o ${CMAKE_CURRENT_BINARY_DIR}
    DEPENDS ${compilecmd}
)

add_library(${STATICLIB} ${sointuasm})
set_target_properties(${STATICLIB} PROPERTIES LINKER_LANGUAGE C)
target_include_directories(${STATICLIB} INTERFACE ${CMAKE_CURRENT_BINARY_DIR})

# Examples are now available.
add_subdirectory(examples)

# Testing only available if this is the main app
# Emergency override 4KLANG_CMAKE_BUILD_TESTING provided as well
if((CMAKE_PROJECT_NAME STREQUAL PROJECT_NAME OR SOINTU_CMAKE_BUILD_TESTING) AND BUILD_TESTING)
    add_subdirectory(tests)
endif()
