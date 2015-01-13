set(TARGET_NAME evmcc)

set(SOURCES
	evmcc.cpp
)
source_group("" FILES ${SOURCES})

add_executable(${TARGET_NAME} ${SOURCES})
set_property(TARGET ${TARGET_NAME} PROPERTY FOLDER "tools")

include_directories(../..)
include_directories(${LLVM_INCLUDE_DIRS})
include_directories(${Boost_INCLUDE_DIRS})

target_link_libraries(${TARGET_NAME} ethereum)
target_link_libraries(${TARGET_NAME} ${Boost_PROGRAM_OPTIONS_LIBRARIES})

install(TARGETS ${TARGET_NAME} DESTINATION bin )