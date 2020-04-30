file(GLOB SOURCE_LIST CONFIGURE_DEPENDS "*.cpp" "*.hpp" "*.asm" "*.bin")
file(GLOB AEFFECT_SOURCE_LIST CONFIGURE_DEPENDS "audioeffect/*")

# Make a dynamic library of 4klang
add_library(4klang SHARED ${SOURCE_LIST} ${AEFFECT_SOURCE_LIST} Go4kVSTi.rc Go4kVSTi.def)

# Make a dynamic library of 8klang
add_library(8klang SHARED ${SOURCE_LIST} ${AEFFECT_SOURCE_LIST} Go4kVSTi2.rc Go4kVSTi2.def)

target_compile_definitions(8klang PUBLIC _8KLANG)

set_target_properties(4klang 8klang PROPERTIES
  MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")

# The private headers 
target_include_directories(4klang PRIVATE "audioeffect/")

# The private headers 
target_include_directories(8klang PRIVATE "audioeffect/")

add_library(vstgui STATIC IMPORTED)
set_target_properties(vstgui
                      PROPERTIES IMPORTED_LOCATION
                      "${CMAKE_SOURCE_DIR}/extern/vstgui.lib")

target_link_libraries(4klang PRIVATE comctl32 vstgui)

target_link_libraries(8klang PRIVATE comctl32 vstgui)
