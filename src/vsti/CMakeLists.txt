file(GLOB SOURCE_LIST CONFIGURE_DEPENDS "*.cpp" "*.hpp" "*.asm" "*.rc" "*.bin")
file(GLOB AEFFECT_SOURCE_LIST CONFIGURE_DEPENDS "audioeffect/*")

# Make an automatic library - will be static or dynamic based on user setting
add_library(4klang SHARED ${SOURCE_LIST} ${AEFFECT_SOURCE_LIST} Go4kVSTi.def)

set_property(TARGET 4klang PROPERTY
  MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")

# The private headers 
target_include_directories(4klang PRIVATE "audioeffect/")

add_library(vstgui STATIC IMPORTED)
set_target_properties(vstgui
                      PROPERTIES IMPORTED_LOCATION
                      "${CMAKE_SOURCE_DIR}/extern/vstgui.lib")

target_link_libraries(4klang PRIVATE comctl32 vstgui)
# add_dependencies(4klang_vsti vstgui)

# This depends on (header only) boost
# target_link_libraries(modern_library PRIVATE Boost::boost)

# All users of this library will need at least C++11
#target_compile_features(modern_library PUBLIC cxx_std_11)

# IDEs should put the headers in a nice place
#source_group(TREE "${PROJECT_SOURCE_DIR}/include" PREFIX "Header Files" #FILES ${HEADER_LIST})
