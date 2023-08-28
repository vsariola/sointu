# identifier: Name of the example
# songfile: File path of the song YAML file.
# architecture: 386 or amd64
# abi: 32 or 64
# windows_libraries: All libraries that you need to link on Windows
# unix_libraries: All libraries that you need to link on unix
function(add_asm_example identifier songfile architecture sizeof_void_ptr windows_libraries unix_libraries)
	get_filename_component(songprefix ${songfile} NAME_WE)

	# Generate the song assembly file
	add_custom_command(
		COMMAND
			${compilecmd} -arch=${architecture} -o ${songprefix}_${architecture}.asm ${songfile}
		WORKING_DIRECTORY
			${CMAKE_CURRENT_BINARY_DIR}
		DEPENDS
			${songfile}
		OUTPUT
			${songprefix}_${architecture}.asm
			${songprefix}_${architecture}.h
			${songprefix}_${architecture}.inc
		COMMENT
			"Compiling ${PROJECT_SOURCE_DIR}/examples/patches/physics-girl-st.yml..."
	)

	# Platform dependent options
	if(WIN32)
		set(abi win)
		set(libraries ${windows_libraries})
		if (CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
			set(link_options -nostartfiles)
		endif()
	elseif(UNIX)
		set(abi elf)
		set(link_options -z noexecstack -no-pie)
		set(libraries ${unix_libraries})
	endif()

	# Add target
	add_executable(${identifier}-${architecture}
		${identifier}.${abi}${sizeof_void_ptr}.asm
		${songprefix}_${architecture}.asm
		${songprefix}_${architecture}.inc
	)
	set_target_properties(${identifier}-${architecture} PROPERTIES ASM_NASM_COMPILE_OPTIONS -f${abi}${sizeof_void_ptr})
	target_include_directories(${identifier}-${architecture} PRIVATE ${CMAKE_CURRENT_BINARY_DIR})
	set_target_properties(${identifier}-${architecture} PROPERTIES LINKER_LANGUAGE C)
	target_link_options(${identifier}-${architecture} PRIVATE -m${sizeof_void_ptr} ${link_options})
	target_link_libraries(${identifier}-${architecture} PRIVATE ${libraries})
	target_compile_definitions(${identifier}-${architecture} PRIVATE TRACK_INCLUDE="${songprefix}_${architecture}.inc")

	# Set up dependencies
	add_dependencies(${identifier}-${architecture} sointu-compiler)
	add_dependencies(examples ${identifier}-${architecture})
endfunction()

add_subdirectory(386)
add_subdirectory(amd64)
