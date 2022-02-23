# cmake_minimum_required(VERSION 3.19)

project(Greeting C CXX)
set (CMAKE_CXX_STANDARD 11)

find_package(Threads REQUIRED)

# find protobuf installation
find_package(Protobuf CONFIG REQUIRED)
message(STATUS "Using protobuf ${Protobuf_VERSION}")
set(_PROTOBUF_LIBPROTOBUF protobuf::libprotobuf)
if(CMAKE_CROSSCOMPILING)
  find_program(_PROTOBUF_PROTOC protoc)
else()
  set(_PROTOBUF_PROTOC $<TARGET_FILE:protobuf::protoc>)
endif()

# Find gRPC installation
find_package(gRPC CONFIG REQUIRED)
message(STATUS "Using gRPC ${gRPC_VERSION}")
set(_REFLECTION gRPC::grpc++_reflection)
set(_GRPC_GRPCPP gRPC::grpc++)
if(CMAKE_CROSSCOMPILING)
  find_program(_GRPC_CPP_PLUGIN_EXECUTABLE grpc_cpp_plugin)
else()
  set(_GRPC_CPP_PLUGIN_EXECUTABLE $<TARGET_FILE:gRPC::grpc_cpp_plugin>)
endif()

# Proto file
get_filename_component(greet_proto "proto/messages.proto" ABSOLUTE)
get_filename_component(greet_proto_path "${greet_proto}" PATH)

# Generated sources
set(greet_proto_srcs "${CMAKE_CURRENT_BINARY_DIR}/messages.pb.cc")
set(greet_proto_hdrs "${CMAKE_CURRENT_BINARY_DIR}/messages.pb.h")
set(greet_grpc_srcs "${CMAKE_CURRENT_BINARY_DIR}/messages.grpc.pb.cc")
set(greet_grpc_hdrs "${CMAKE_CURRENT_BINARY_DIR}/messages.grpc.pb.h")
add_custom_command(
      OUTPUT "${greet_proto_srcs}" "${greet_proto_hdrs}" "${greet_grpc_srcs}" "${greet_grpc_hdrs}"
      COMMAND ${_PROTOBUF_PROTOC}
      ARGS --grpc_out "${CMAKE_CURRENT_BINARY_DIR}"
        --cpp_out "${CMAKE_CURRENT_BINARY_DIR}"
        -I "${greet_proto_path}"
        --plugin=protoc-gen-grpc="${_GRPC_CPP_PLUGIN_EXECUTABLE}"
        "${greet_proto}"
      DEPENDS "${greet_proto}")

# Include generated *.pb.h files
include_directories("${CMAKE_CURRENT_BINARY_DIR}")

include_directories("otelgrpc")
include_directories("hindsightgrpc")

# greet_grpc_proto
add_library(greet_grpc_proto
  ${greet_grpc_srcs}
  ${greet_grpc_hdrs}
  ${greet_proto_srcs}
  ${greet_proto_hdrs})
target_link_libraries(greet_grpc_proto
  ${_REFLECTION}
  ${_GRPC_GRPCPP}
  ${_PROTOBUF_LIBPROTOBUF})

# find opentelemetry installation
find_package(opentelemetry-cpp CONFIG REQUIRED)
message(STATUS "Using opentelemetry-cpp ${OPENTELEMETRY_VERSION}")

# Targets greeter_[async_](client|server)
foreach(_target
  greeter_client greeter_server)
  add_executable(${_target} "${_target}.cc")
  target_link_libraries(${_target}
    greet_grpc_proto
    ${_REFLECTION}
    ${_GRPC_GRPCPP}
    # note that the order matters, put caller before callee
    opentelemetry_trace
    opentelemetry_exporter_ostream_span
    opentelemetry_exporter_jaeger_trace
    opentelemetry_exporter_otlp_http    
    opentelemetry_exporter_otlp_http_client
    opentelemetry_otlp_recordable
    opentelemetry_proto
    opentelemetry_common
    opentelemetry_metrics
    opentelemetry_version
    opentelemetry_resources
    # build the following libraries and add them to the path
    ${_PROTOBUF_LIBPROTOBUF}
    http_client_curl
    thrift
    curl
    )
endforeach()
