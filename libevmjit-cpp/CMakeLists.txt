
project(evmjit-cpp LANGUAGES CXX)

set(SOURCES
    Ext.cpp     Ext.h
    Runtime.cpp Runtime.h
    VM.cpp      VM.h
)

source_group("" FILES ${SOURCES})

add_library(${PROJECT_NAME} ${SOURCES})
set_property(TARGET ${PROJECT_NAME} PROPERTY FOLDER "libs")

