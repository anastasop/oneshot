cmake_minimum_required(VERSION 3.10)

project(hello)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_STANDARD_REQUIRED True)
set(ABSL_PROPAGATE_CXX_STD ON)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

add_subdirectory(src/leveldb)
add_subdirectory(src/abseil-cpp)

add_executable(hello src/hello.cc)

target_link_libraries(hello PUBLIC leveldb)
target_link_libraries(hello PUBLIC abseil-cpp)
