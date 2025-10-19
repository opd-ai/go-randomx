cmake_minimum_required(VERSION 3.10)
project(RandomXTraceExtractor)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find RandomX library
find_library(RANDOMX_LIB randomx)
if(NOT RANDOMX_LIB)
    message(FATAL_ERROR "RandomX library not found. Please install RandomX first:")
    message(FATAL_ERROR "  git clone https://github.com/tevador/RandomX.git")
    message(FATAL_ERROR "  cd RandomX && mkdir build && cd build")
    message(FATAL_ERROR "  cmake -DARCH=native .. && make && sudo make install")
endif()

# Find RandomX headers
find_path(RANDOMX_INCLUDE randomx.h
    PATHS
    /usr/local/include
    /usr/include
    ${CMAKE_SOURCE_DIR}/../../RandomX/src
)
if(NOT RANDOMX_INCLUDE)
    message(FATAL_ERROR "RandomX headers not found")
endif()

include_directories(${RANDOMX_INCLUDE})

add_executable(extract_trace extract_trace.cpp)
target_link_libraries(extract_trace ${RANDOMX_LIB})

message(STATUS "RandomX library: ${RANDOMX_LIB}")
message(STATUS "RandomX include: ${RANDOMX_INCLUDE}")
