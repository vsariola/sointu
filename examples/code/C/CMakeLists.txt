# this fixes a bug in creating a static library from asm, similar to
# https://discourse.cmake.org/t/building-lib-file-from-asm-cmake-bug/1959
# but for NASM
if(MSVC)
    set(CMAKE_ASM_NASM_CREATE_STATIC_LIBRARY "<CMAKE_AR> /OUT:<TARGET> <LINK_FLAGS> <OBJECTS>")
endif()

add_custom_command(
	COMMAND
		${compilecmd} -arch=${arch} -o physics_girl_st.asm "${PROJECT_SOURCE_DIR}/examples/patches/physics_girl_st.yml"
	WORKING_DIRECTORY
		${CMAKE_CURRENT_BINARY_DIR}
	DEPENDS
		"${PROJECT_SOURCE_DIR}/examples/patches/physics_girl_st.yml"
		${compilecmd}
	OUTPUT
		physics_girl_st.asm
		physics_girl_st.h
		physics_girl_st.inc
	COMMENT
		"Compiling ${PROJECT_SOURCE_DIR}/examples/patches/physics-girl-st.yml..."
)

add_library(physics_girl_st physics_girl_st.asm)

if(WIN32)
	add_executable(cplay-winmm
		cplay.windows.winmm.c
		physics_girl_st.h
	)
	target_link_libraries(cplay-winmm PRIVATE winmm)
	target_link_libraries(cplay-winmm PRIVATE physics_girl_st)
	target_include_directories(cplay-winmm PRIVATE ${CMAKE_CURRENT_BINARY_DIR})
	add_dependencies(examples cplay-winmm)

	add_executable(cplay-directsound
		cplay.windows.directsound.c
		physics_girl_st.h
	)
	target_link_libraries(cplay-directsound PRIVATE dsound ws2_32 ucrt)
	target_link_libraries(cplay-directsound PRIVATE physics_girl_st)
	target_include_directories(cplay-directsound PRIVATE ${CMAKE_CURRENT_BINARY_DIR})
	add_dependencies(examples cplay-directsound)
elseif(UNIX)
	add_executable(cplay
		cplay.unix.c
		physics_girl_st.h
	)
	target_link_libraries(cplay PRIVATE asound pthread)
	target_link_options(cplay PRIVATE -z noexecstack -no-pie)
	target_link_libraries(cplay PRIVATE physics_girl_st)
	target_include_directories(cplay PRIVATE ${CMAKE_CURRENT_BINARY_DIR})
	add_dependencies(examples cplay)
endif()

add_executable(cwav
	cwav.c
	physics_girl_st.h
)
if(WIN32)
	target_compile_definitions(cwav PRIVATE _CRT_SECURE_NO_WARNINGS)
elseif(UNIX)
	target_link_options(cwav PRIVATE -z noexecstack -no-pie)
endif()
target_link_libraries(cwav PRIVATE physics_girl_st)
target_include_directories(cwav PRIVATE ${CMAKE_CURRENT_BINARY_DIR})

add_dependencies(examples cwav)
