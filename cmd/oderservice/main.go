package main

import (
	"fmt"
	"log"
	"net"

	grpc_server "go-grpc-server/internal/app/adapter/grpc-server"
	pb "go-grpc-server/internal/app/protos/orderservice"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50051))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterOrderServiceServer(s, grpc_server.NewServer())

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
