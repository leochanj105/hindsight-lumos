cmake_minimum_required(VERSION 3.5.1)

project(HindsightBenchmarks C CXX)

set (CMAKE_CXX_STANDARD 11)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

set(CMAKE_CXX_FLAGS_DEBUG "-g")
set(CMAKE_CXX_FLAGS_RELEASE "-O3")

find_package(Threads REQUIRED)

# Hindsight include
#include_directories(/usr/local/include/tracer/)
include_directories(../client/src/ ../client/include/)

# Source directory
file (GLOB CPP_FILES src/hindsight/*.cc) 
file (GLOB H_FILES src/hindsight/*.h) 
set (SOURCE_FILES ${CPP_FILES} ${H_FILES})

# Targets greeter_[async_](client|server)
foreach(_target
  benchmark test_autotriggers)
  add_executable(${_target} "src/${_target}.cc" ${CPP_FILES})
  target_include_directories(${_target} PRIVATE)
  target_link_libraries(${_target} Threads::Threads ${CMAKE_SOURCE_DIR}/../client/lib/libtracer.a)
endforeach()
