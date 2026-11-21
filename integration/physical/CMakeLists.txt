cmake_minimum_required(VERSION 3.21)
project(EssieuPilotATS LANGUAGES C CXX)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if(NOT DEFINED TS_EXTRA_ROOT OR TS_EXTRA_ROOT STREQUAL "")
  message(FATAL_ERROR "TS_EXTRA_ROOT must point to the pinned MIT source checkout (for MinHook + SCS SDK)")
endif()
add_subdirectory("${TS_EXTRA_ROOT}/vendor/minhook" "${CMAKE_CURRENT_BINARY_DIR}/minhook-build")
set_property(TARGET minhook PROPERTY MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
add_library(EssieuPilotATS SHARED EssieuPilotATS.cpp)
target_compile_features(EssieuPilotATS PRIVATE cxx_std_17)
target_include_directories(EssieuPilotATS PRIVATE
  "${TS_EXTRA_ROOT}/vendor/scs_sdk/include"
  "${TS_EXTRA_ROOT}/vendor/minhook/include")
target_link_libraries(EssieuPilotATS PRIVATE minhook bcrypt)
set_target_properties(EssieuPilotATS PROPERTIES
  OUTPUT_NAME "EssieuPilotATS" MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
