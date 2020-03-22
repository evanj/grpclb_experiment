package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/grpclb/grpc_lb_v1"
	"google.golang.org/grpc/metadata"
)

const clientStatsInterval = 10 * time.Second

type balancer struct {
	backendIP   []byte
	backendPort int
}

func (b *balancer) BalanceLoad(stream grpc_lb_v1.LoadBalancer_BalanceLoadServer) error {
	ctx := stream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	log.Printf("BalanceLoad stream starting md=%v ok=%v ...", md, ok)
	for {
		req, err := stream.Recv()
		log.Printf("BalanceLoad Recv()=%s err=%v", req.String(), err)
		if err != nil {
			return err
		}

		if req.GetInitialRequest() != nil {
			log.Printf("got initial request")

			// send the initial response then the server list
			resp := &grpc_lb_v1.LoadBalanceResponse{}
			resp.LoadBalanceResponseType = &grpc_lb_v1.LoadBalanceResponse_InitialResponse{
				InitialResponse: &grpc_lb_v1.InitialLoadBalanceResponse{
					ClientStatsReportInterval: ptypes.DurationProto(clientStatsInterval),
				},
			}
			err = stream.Send(resp)
			if err != nil {
				log.Println("Send return an error", err)
				return err
			}

			resp.LoadBalanceResponseType = &grpc_lb_v1.LoadBalanceResponse_ServerList{
				ServerList: &grpc_lb_v1.ServerList{
					Servers: []*grpc_lb_v1.Server{
						&grpc_lb_v1.Server{
							IpAddress:        b.backendIP,
							Port:             int32(b.backendPort),
							LoadBalanceToken: "backend1",
						},
					},
				},
			}
			err = stream.Send(resp)
			if err != nil {
				log.Println("Send return an error", err)
				return err
			}

			log.Println("sent initial response and server list")
		}
	}
}

func main() {
	grpcAddr := flag.String("grpcAddr", "localhost:9001", "Address to listen for gRPC requests")
	backend := flag.String("backend", "localhost:8081", "Address for the client backend")
	flag.Parse()

	resolved, err := net.ResolveTCPAddr("tcp", *backend)
	if err != nil {
		panic(err)
	}
	log.Printf("resolved backend=%s to %s:%d", *backend, resolved.IP.String(), resolved.Port)

	balancerServer := &balancer{resolved.IP, resolved.Port}

	log.Printf("listening for gRPC on grpcAddr=%s ...", *grpcAddr)
	grpcListener, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	grpc_lb_v1.RegisterLoadBalancerServer(grpcServer, balancerServer)
	err = grpcServer.Serve(grpcListener)
	if err != nil {
		panic(err)
	}
}
