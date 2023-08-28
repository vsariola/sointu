add_custom_command(
    COMMAND
        ${compilecmd} -arch=${arch} -o physics_girl_st.asm "${PROJECT_SOURCE_DIR}/examples/patches/physics_girl_st.yml"
    WORKING_DIRECTORY
        ${CMAKE_CURRENT_BINARY_DIR}
    DEPENDS
        "${PROJECT_SOURCE_DIR}/examples/patches/physics_girl_st.yml"
    OUTPUT
        physics_girl_st.asm
        physics_girl_st.h
        physics_girl_st.inc
    COMMENT
        "Compiling ${PROJECT_SOURCE_DIR}/examples/patches/physics-girl-st.yml..."
)

add_library(physics_girl_st physics_girl_st.asm)
add_dependencies(physics_girl_st sointu-compiler)

if(WIN32)
    add_executable(cplay
        cplay.windows.c
        physics_girl_st.h
    )
    target_link_libraries(cplay PRIVATE winmm)
elseif(UNIX)
    add_executable(cplay
        cplay.unix.c
        physics_girl_st.h
    )
    target_link_libraries(cplay PRIVATE asound pthread)
    target_link_options(cplay PRIVATE -z noexecstack -no-pie)
endif()
target_link_libraries(cplay PRIVATE physics_girl_st)
target_include_directories(cplay PRIVATE ${CMAKE_CURRENT_BINARY_DIR})
