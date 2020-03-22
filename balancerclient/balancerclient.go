package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"google.golang.org/grpc/resolver"

	_ "google.golang.org/grpc/balancer/grpclb"
	"google.golang.org/grpc/grpclog"

	"github.com/evanj/concurrentlimit/sleepymemory"
	"github.com/evanj/grpclb_experiment/grpc_service_config"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

const grpcConnectTimeout = 30 * time.Second

// func sendRequestsGoroutine(
// 	done <-chan struct{}, totalRequestsChan chan<- int, sender requestSender,
// 	req *sleepymemory.SleepRequest,
// ) {
// 	// create a new sender for each goroutine
// 	sender = sender.clone()

// 	requestCount := 0
// sendLoop:
// 	for {
// 		// if done is closed, break out of the loop
// 		select {
// 		case <-done:
// 			break sendLoop
// 		default:
// 		}

// 		err := sender.send(req)
// 		if err != nil {
// 			if err == errRetry || err == context.DeadlineExceeded {
// 				// TODO: exponential backoff?
// 				time.Sleep(time.Second)
// 				continue
// 			}
// 			panic(err)
// 		}

// 		requestCount++
// 	}
// 	totalRequestsChan <- requestCount
// }

const staticResolverScheme = "static_do_not_register"
const grpcSchemeSeparator = ":///"

type staticBalancerResolverBuilder struct {
	grpclbBalancer string
}

func (b *staticBalancerResolverBuilder) Build(
	target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions,
) (resolver.Resolver, error) {
	log.Println("static resolver builder Build() called")
	r := &staticResolver{target, cc, b.grpclbBalancer}
	r.start()
	return r, nil
}
func (*staticBalancerResolverBuilder) Scheme() string { return staticResolverScheme }

type staticResolver struct {
	target         resolver.Target
	cc             resolver.ClientConn
	grpclbBalancer string
}

func (r *staticResolver) start() {
	addrs := []resolver.Address{
		resolver.Address{
			Addr: r.grpclbBalancer,
			// the docs say this is deprecated, but this is what the dns resolver actually does
			Type: resolver.GRPCLB,
		},
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*staticResolver) ResolveNow(o resolver.ResolveNowOptions) {
	log.Println("staticResolver ResolveNow()")
}
func (*staticResolver) Close() {
	log.Println("staticResolver Close()()")
}

func main() {
	grpcTarget := flag.String("grpcTarget", "localhost:8081", "gRPC address of the real backend server")
	balancer := flag.String("balancer", "localhost:9001", "gRPC address of the grpclb load balancer")
	sleep := flag.Duration("sleep", time.Second, "Time for the server to sleep handling a request")
	flag.Parse()

	logg := grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(logg)

	req := &sleepymemory.SleepRequest{
		SleepDuration: ptypes.DurationProto(*sleep),
	}

	// create a ServiceConfig protobuf struct manually
	// TODO: figure out how to do this programatically in an API that is not deprecated
	lbConfig := &grpc_service_config.LoadBalancingConfig{
		Policy: &grpc_service_config.LoadBalancingConfig_Grpclb{
			Grpclb: &grpc_service_config.GrpcLbConfig{
				// ChildPolicy: []*grpc_service_config.LoadBalancingConfig{{Policy: &grpc_service_config.LoadBalancingConfig_RoundRobin{
				// 	RoundRobin: &grpc_service_config.RoundRobinConfig{},
				// }}},
			},
		},
	}
	defaultConfig := &grpc_service_config.ServiceConfig{
		LoadBalancingConfig: []*grpc_service_config.LoadBalancingConfig{lbConfig},
	}
	configBuf := &bytes.Buffer{}
	err := (&jsonpb.Marshaler{}).Marshal(configBuf, defaultConfig)
	if err != nil {
		panic(err)
	}
	serviceConfig := configBuf.String()
	log.Printf("Creating default service config=%s", serviceConfig)

	// create a resolver that will return balancer as the load balancer to use.
	// See: https://github.com/grpc/grpc-go/blob/master/examples/features/name_resolving/README.md
	configuredResolver := &staticBalancerResolverBuilder{*balancer}

	fullyQualifiedName := staticResolverScheme + grpcSchemeSeparator + *grpcTarget
	log.Printf("Calling gRPC Dial(%s) ...", fullyQualifiedName)
	conn, err := grpc.Dial(fullyQualifiedName,
		grpc.WithInsecure(),
		grpc.WithTimeout(grpcConnectTimeout),
		grpc.WithBlock(),
		// grpc.WithDefaultServiceConfig(serviceConfig),
		grpc.WithResolvers(configuredResolver),
	)
	if err != nil {
		panic(err)
	}

	client := sleepymemory.NewSleeperClient(conn)
	for i := 0; i < 60; i++ {
		resp, err := client.Sleep(context.Background(), req)
		log.Printf("response=%s err=%v", resp.String(), err)
	}

	// log.Printf("sending requests for %s using %d client goroutines ...",
	// 	duration.String(), *concurrent)
	// done := make(chan struct{})
	// totalRequestsChan := make(chan int)
	// for i := 0; i < *concurrent; i++ {
	// 	go sendRequestsGoroutine(done, totalRequestsChan, sender, req)
	// }

	// time.Sleep(*duration)
	// close(done)

	// totalRequests := 0
	// for i := 0; i < *concurrent; i++ {
	// 	totalRequests += <-totalRequestsChan
	// }
	// close(totalRequestsChan)

	// log.Printf("sent %d requests in %s using %d clients = %.3f reqs/sec",
	// 	totalRequests, duration.String(), *concurrent, float64(totalRequests)/duration.Seconds())
}
