#include <iostream>
#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>

#include "opentelemetry/trace/context.h"
#include "opentelemetry/trace/experimental_semantic_conventions.h"
#include "opentelemetry/trace/span_context_kv_iterable_view.h"

#include "messages.grpc.pb.h"
#include "otelgrpc/otel_tracer_common.h"

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::ServerWriter;
using grpc::Status;

using namespace opentelemetry::trace;
using Span = opentelemetry::trace::Span;
using SpanContext = opentelemetry::trace::SpanContext;
namespace Context = opentelemetry::context;

using greeting::Greeter;
using greeting::GreetRequest;
using greeting::GreetResponse;

class GreeterServiceImpl final : public Greeter::Service {
  Status Greet(ServerContext *context, const GreetRequest *request,
               GreetResponse *response) override {

#ifdef TRACING
    // setting up the tracer
    StartSpanOptions options;
    // options.kind = SpanKind::kServer;

    // extract context from grpc call metadata and set the span to a new context
    auto prop =
        Context::propagation::GlobalTextMapPropagator::GetGlobalPropagator();
    GrpcServerCarrier carrier(context);
    auto current_ctx = Context::RuntimeContext::GetCurrent();
    auto new_context = prop->Extract(carrier, current_ctx);
    options.parent = GetSpan(new_context)->GetContext();

    // auto trace_parent = carrier.Get("traceparent");
    // std::cout << "trace parent " << trace_parent << "\n";

    // auto span_context =
    //     opentelemetry::v1::nostd::get<trace_api::SpanContext>(options.parent);
    // assert(span_context.IsValid());

    std::string span_name = "GreeterService/Greet";
    // no extra attributes are specified to reduce network traffic
    auto span = get_tracer("grpc")->StartSpan(span_name, options);
    // mark the span as active, not useful here though
    auto scope = get_tracer("grpc")->WithActiveSpan(span);

    span->AddEvent("Processing client attributes");
#endif

    float *values = new float[10000];

    float x = 5.0;
    for (int i = 0; i < 100000; i++) {
      values[i * 9999 % 10000] = (x *= 1.156);
    }

    delete values;

    // the actual RPC response sent to the client
    std::string prefix("Hello ");
    response->set_response(prefix + request->request());

#ifdef TRACING
    // more span events and then end the span
    span->AddEvent("Response sent to client");
    span->SetStatus(StatusCode::kOk);
    span->End();
#endif

    return Status::OK;
  }
};

void RunServer(std::string server_addr) {
  GreeterServiceImpl service;
  ServerBuilder builder;

  builder.RegisterService(&service);
  builder.AddListeningPort(server_addr, grpc::InsecureServerCredentials());

  // Finally assemble the server.
  std::unique_ptr<Server> server(builder.BuildAndStart());
  std::cout << "Server listening on " << server_addr << std::endl;

  // Wait for the server to shutdown. Note that some other thread must be
  // responsible for shutting down the server for this call to ever return.
  server->Wait();
}

int main(int argc, char **argv) {

  std::string server_addr = "0.0.0.0:8800";
  std::string collector_ip = "0.0.0.0";
  int collector_port = 6832;

  if (argc == 1) {
    std::cout << "./client [server_addr] [collector_ip] [collector_port]"
              << std::endl;
  } else {
    if (argc > 1)
      server_addr = argv[1];
    if (argc > 2)
      collector_ip = argv[2];
    if (argc > 3)
      collector_port = atoi(argv[3]);
  }

#ifdef TRACING
  initTracer(collector_ip, collector_port);
#endif

  RunServer(server_addr);

  return 0;
}
