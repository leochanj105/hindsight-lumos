// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

#pragma once

#include <cstring>
#include <iostream>
#include <vector>

#include <grpcpp/grpcpp.h>

#include "opentelemetry/context/propagation/global_propagator.h"
#include "opentelemetry/context/propagation/text_map_propagator.h"
#include "opentelemetry/exporters/jaeger/jaeger_exporter.h"
#include "opentelemetry/exporters/ostream/span_exporter.h"
#include "opentelemetry/exporters/otlp/otlp_http_exporter.h"
#include "opentelemetry/nostd/shared_ptr.h"
#include "opentelemetry/sdk/trace/simple_processor.h"
#include "opentelemetry/sdk/trace/tracer_provider.h"
#include "opentelemetry/trace/propagation/http_trace_context.h"
#include "opentelemetry/trace/provider.h"

#define TRACING 1

using grpc::ClientContext;
using grpc::ServerContext;

class GrpcClientCarrier
    : public opentelemetry::context::propagation::TextMapCarrier {
public:
  GrpcClientCarrier(ClientContext *context) : context_(context) {}
  GrpcClientCarrier() = default;
  virtual opentelemetry::nostd::string_view
  Get(opentelemetry::nostd::string_view key) const noexcept override {
    return "";
  }

  virtual void Set(opentelemetry::nostd::string_view key,
                   opentelemetry::nostd::string_view value) noexcept override {
    // std::cout << " Client ::: Adding " << key << " " << value << "\n";
    context_->AddMetadata(key.data(), value.data());
  }

  ClientContext *context_;
};

class GrpcServerCarrier
    : public opentelemetry::context::propagation::TextMapCarrier {
public:
  GrpcServerCarrier(ServerContext *context) : context_(context) {}
  GrpcServerCarrier() = default;
  virtual opentelemetry::nostd::string_view
  Get(opentelemetry::nostd::string_view key) const noexcept override {
    auto it = context_->client_metadata().find(key.data());
    if (it != context_->client_metadata().end()) {
      // std::cout << " Server ::: Accessing " << key << " " <<
      // it->second.data() << "\n";
      return it->second.data();
    }
    return "";
  }

  virtual void Set(opentelemetry::nostd::string_view key,
                   opentelemetry::nostd::string_view value) noexcept override {
    // Not required for server
  }

  ServerContext *context_;
};

void initTracer(std::string exporter_ip, int exporter_port) {
  // direct the traces to the output stream
  // auto exporter = std::unique_ptr<opentelemetry::sdk::trace::SpanExporter>(
  //     new opentelemetry::exporter::trace::OStreamSpanExporter);

  // Create Jaeger exporter instance
  opentelemetry::exporter::jaeger::JaegerExporterOptions opts;
  opts.endpoint = exporter_ip;
  opts.server_port = exporter_port;
  opts.transport_format =
      opentelemetry::exporter::jaeger::TransportFormat::kThriftUdpCompact;
  auto exporter = std::unique_ptr<opentelemetry::sdk::trace::SpanExporter>(
      new opentelemetry::exporter::jaeger::JaegerExporter(opts));

  // Create OTLP http exporter instance
  // opentelemetry::exporter::otlp::OtlpHttpExporterOptions opts;
  // opts.url = exporter_ip + std::string(":") + std::to_string(exporter_port);
  // auto exporter =
  // std::unique_ptr<opentelemetry::sdk::trace::SpanExporter>(new
  // opentelemetry::exporter::otlp::OtlpHttpExporter(opts));

  auto processor = std::unique_ptr<opentelemetry::sdk::trace::SpanProcessor>(
      new opentelemetry::sdk::trace::SimpleSpanProcessor(std::move(exporter)));
  std::vector<std::unique_ptr<opentelemetry::sdk::trace::SpanProcessor>>
      processors;
  processors.push_back(std::move(processor));
  // Default is an always-on sampler.
  auto context = std::make_shared<opentelemetry::sdk::trace::TracerContext>(
      std::move(processors));
  auto provider =
      opentelemetry::nostd::shared_ptr<opentelemetry::trace::TracerProvider>(
          new opentelemetry::sdk::trace::TracerProvider(context));
  // Set the global trace provider
  opentelemetry::trace::Provider::SetTracerProvider(provider);

  // set global propagator
  opentelemetry::context::propagation::GlobalTextMapPropagator::
      SetGlobalPropagator(
          opentelemetry::nostd::shared_ptr<
              opentelemetry::context::propagation::TextMapPropagator>(
              new opentelemetry::trace::propagation::HttpTraceContext()));
}

opentelemetry::nostd::shared_ptr<opentelemetry::trace::Tracer>
get_tracer(std::string tracer_name) {
  auto provider = opentelemetry::trace::Provider::GetTracerProvider();
  return provider->GetTracer(tracer_name);
}
