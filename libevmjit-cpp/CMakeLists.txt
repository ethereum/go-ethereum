
project(evmjit-cpp LANGUAGES CXX)

set(SOURCES VM.cpp VM.h)

source_group("" FILES ${SOURCES})

add_library(${PROJECT_NAME} ${SOURCES})
set_property(TARGET ${PROJECT_NAME} PROPERTY FOLDER "libs")

