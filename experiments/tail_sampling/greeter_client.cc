#include <iostream>
#include <memory>
#include <pthread.h>
#include <string>

#include <grpcpp/grpcpp.h>

#include "opentelemetry/trace/experimental_semantic_conventions.h"

#include "messages.grpc.pb.h"
#include "otelgrpc/otel_tracer_common.h"

using grpc::Channel;
using grpc::ClientContext;
using grpc::Status;

using namespace opentelemetry::trace;
namespace Context = opentelemetry::context;

using greeting::Greeter;
using greeting::GreetRequest;
using greeting::GreetResponse;

class GreeterClient {
public:
  GreeterClient(std::shared_ptr<Channel> channel)
      : stub_(Greeter::NewStub(channel)) {}

  std::string Greet(const std::string &greeting, const int thread_id,
                    const int request_id) {
    // setting up a rpc request
    GreetRequest request;
    GreetResponse response;
    // Todo: figured out the problem of context propagation
    // Todo: switch to propagate information within gprc message for fair
    ClientContext context;
    request.set_request(greeting);

#ifdef TRACING
    // setting up the tracer
    StartSpanOptions options;
    // options.kind = SpanKind::kClient;
    std::string span_name = "GreeterClient/GreetFromThread#" +
                            std::to_string(thread_id) + "Request#" +
                            std::to_string(request_id);
    // no extra attributes are specified to reduce network traffic
    auto span = get_tracer("grpc")->StartSpan(span_name, options);

    // mark the span as active
    auto scope = get_tracer("grpc-client")->WithActiveSpan(span);

    // inject information of the context (with active span) into the RPC call
    auto current_ctx = Context::RuntimeContext::GetCurrent();
    GrpcClientCarrier carrier(&context);
    auto prop =
        Context::propagation::GlobalTextMapPropagator::GetGlobalPropagator();
    prop->Inject(carrier, current_ctx);
#endif

    // The actual RPC request sent to the server
    Status status = stub_->Greet(&context, request, &response);

    // Act upon its status.
    if (status.ok()) {
#ifdef TRACING
      span->SetStatus(StatusCode::kOk);
      // span->SetAttribute(OTEL_GET_TRACE_ATTR(AttrRpcGrpcStatusCode),
      //                    status.error_code());
      span->End();
#endif
      return response.response();
    } else {
#ifdef TRACING
      std::cout << status.error_code() << ": " << status.error_message()
                << std::endl;
      span->SetStatus(StatusCode::kError);
      // span->SetAttribute(OTEL_GET_TRACE_ATTR(AttrRpcGrpcStatusCode),
      //                    status.error_code());
      span->End();
#endif
      return "RPC failed";
    }
  }

private:
  std::unique_ptr<Greeter::Stub> stub_;
};

struct client_params {
  int thread_id;
  int num_requests;
  std::string server_addr;
};

void *run_client_thread(void *ptr) {
  client_params *params = (client_params *)ptr;
  GreeterClient greeter(grpc::CreateChannel(
      params->server_addr, grpc::InsecureChannelCredentials()));

  for (int i = 0; i < params->num_requests; i++) {
    std::string greeting("greeting" + std::to_string(i) + " from thread#" +
                         std::to_string(params->thread_id));
    std::string reply = greeter.Greet(greeting, params->thread_id, i);
    // std::cout << "Greeter received: " << reply << std::endl;
  }
  return 0;
}

int main(int argc, char **argv) {

  int num_threads = 8;
  int num_requests = 1000;
  std::string server_addr = "0.0.0.0:8800";
  std::string collector_ip = "0.0.0.0";
  int collector_port = 6831;

  if (argc == 1) {
    std::cout << "./client [num_threads] [num_requests] [server_addr] "
                 "[collector_ip] [collector_port]"
              << std::endl;
  } else {
    if (argc > 1)
      num_threads = atoi(argv[1]);
    if (argc > 2)
      num_requests = atoi(argv[2]);
    if (argc > 3)
      server_addr = argv[3];
    if (argc > 4)
      collector_ip = argv[4];
    if (argc > 5)
      collector_port = atoi(argv[5]);
  }

#ifdef TRACING
  initTracer(collector_ip, collector_port);
#endif

  pthread_t threads[num_threads];
  client_params params[num_threads];
  for (int i = 0; i < num_threads; i++) {
    params[i].thread_id = i;
    params[i].num_requests = num_requests;
    params[i].server_addr = server_addr;
    pthread_create(&threads[i], NULL, run_client_thread, (void *)&params[i]);
  }

  for (int i = 0; i < num_threads; i++) {
    pthread_join(threads[i], NULL);
  }
  printf("Clients complete.\n");

  return 0;
}
