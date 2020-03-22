# gRPC load balancing experiments

gRPC includes a "lookaside" load balancing implementation called grpclb, although it is now deprecated in favour of xDS: https://groups.google.com/forum/#!msg/grpc-io/0yGihF-EFQo/A4QKdXffBwAJ. Instead of the client directly making all decisions, it asks a load balancer about which backends it should talk to, then just uses round-robin load balancing between them. Sadly, it is not well documented, and I suspect extremely rarely used given the lack of information out there. Most people probably use either:

* Send all requests to a single server (default behaviour)
* Spread requests across all backends using This repository contains a small experiment to play with it.

There is also a new experimental load balancer called XDS that is part of the Envoy/Istio projects that are designed to eventually replace grpclb with something that is more broadly supported. See https://github.com/grpc/grpc-go/issues/3286 https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol


## Running the server, load balancer and client all locally

1. Clone the concurrentlimit repository which contains sleepyserver: https://github.com/evanj/concurrentlimit
2. `go run ./sleepyserver --logAll`
3. `go run ./grpcbalancer`
4. `go run ./balancerclient`

This will log lots of details about the requests in progress.



## How this works

* Dial does "name resolution" according to gRPC's naming policy: https://github.com/grpc/grpc/blob/master/doc/naming.md . By default, this will look up the IP address / DNS name. If DNS has SRV records, it can use the grpclb balancer. To override this, the test client creates a manual resolver that returns a single balancer.

* Copied the compiled proto from https://raw.githubusercontent.com/grpc/grpc-go/master/internal/proto/grpc_service_config/service_config.pb.go

## Resources

* gRPC service config docs: https://github.com/grpc/grpc/blob/master/doc/service_config.md
* gRPC service config proto: https://github.com/grpc/grpc-proto/blob/master/grpc/service_config/service_config.proto

* jawlb: An example of using grpclb with a Kubernetes headless service: https://github.com/joa/jawlb