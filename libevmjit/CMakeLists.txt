set(TARGET_NAME evmjit)

file(GLOB SOURCES "*.cpp")
file(GLOB HEADERS "*.h")
source_group("" FILES ${HEADERS})
source_group("" FILES ${SOURCES})

add_library(${TARGET_NAME} ${SOURCES} ${HEADERS})
set_property(TARGET ${TARGET_NAME} PROPERTY FOLDER "libs")

include_directories(${LLVM_INCLUDE_DIRS})
include_directories(${Boost_INCLUDE_DIRS})
add_definitions(${LLVM_DEFINITIONS})

llvm_map_components_to_libnames(LLVM_LIBS core support mcjit x86asmparser x86codegen)
target_link_libraries(${TARGET_NAME} ${LLVM_LIBS})

#install( TARGETS ${EXECUTABLE} ARCHIVE DESTINATION lib LIBRARY DESTINATION lib )
#install( FILES ${HEADERS} DESTINATION include/${EXECUTABLE} )
