if(("${CMAKE_SIZEOF_VOID_P}" STREQUAL "4") AND MSVC)
    # in 32-bit mode with MSVC toolset, we can use Crinkler to compress the executable
    set(CRINKLER_LEVEL "off" CACHE STRING "Crinkler compression level: off, light, medium, heavy")

    if(NOT CRINKLER_LEVEL STREQUAL OFF)
        find_program(CRINKLER NAMES Crinkler)
        if (NOT CRINKLER)
            message(WARNING "Crinkler not found. Cannot compress executable; using default linker. Get Crinkler from https://github.com/runestubbe/Crinkler & put it in path (as Crinkler.exe)")
            set(CRINKLER_LEVEL OFF)
        endif()
    endif()

    if (NOT CRINKLER_LEVEL STREQUAL OFF)
        message(STATUS "Crinkler found at: ${CRINKLER}")
        set(CRINKLER_FLAGS "/PROGRESSGUI /UNSAFEIMPORT /UNALIGNCODE /HASHSIZE:1000 /REPORT:<TARGET>.report.html")
        # TBD: do we add /SATURATE
        if (CRINKLER_LEVEL STREQUAL LIGHT)
            set(CRINKLER_FLAGS "${CRINKLER_FLAGS} /HASHTRIES:100 /COMPMODE:INSTANT /ORDERTRIES:2000")
        elseif (CRINKLER_LEVEL STREQUAL HEAVY)
            set(CRINKLER_FLAGS "${CRINKLER_FLAGS} /HASHTRIES:1000 /COMPMODE:VERYSLOW /ORDERTRIES:30000")
        else()
            set(CRINKLER_FLAGS "${CRINKLER_FLAGS} /HASHTRIES:300 /COMPMODE:SLOW /ORDERTRIES:9000")
        endif()

        # we drop the whole manifest creation from the front; did not find a way to disable it from CMake otherwise
        set (CMAKE_C_LINK_EXECUTABLE "${CRINKLER} <OBJECTS> /out:<TARGET> ${CRINKLER_FLAGS} <LINK_LIBRARIES>")
    endif()
endif()

set_directory_properties(PROPERTIES EXCLUDE_FROM_ALL ON)
add_custom_target(examples)

add_subdirectory(asm)
add_subdirectory(C)
