package grpc_server

import (
	"context"
	"errors"

	pb "go-grpc-server/internal/app/protos/orderservice"
)

type Server struct {
	pb.UnimplementedOrderServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) DeployService(_ context.Context, req *pb.DeployServiceRequest) (*pb.DeployServiceResponse, error) {
	if req == nil {
		return nil, errors.New("bad request - nil DeployServiceRequest")
	}

	deployedServiceIds := make([]string, 0, len(req.ServiceIds))

	for _, id := range req.ServiceIds {
		deployedServiceIds = append(deployedServiceIds, id)
	}

	return &pb.DeployServiceResponse{
		ServiceIds: deployedServiceIds,
	}, nil
}
