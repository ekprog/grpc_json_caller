package grpc_json_caller

import (
	"context"
	"encoding/json"
	pb "github.com/ekprog/grpc_json_caller/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"log"
	"net"
	"testing"
)

func makeGRPCServer(addr string) *grpc.Server {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("GRPC server listening at %v\n", lis.Addr())

	// Registering test server for testing
	pb.RegisterTestServiceServer(grpcServer, &TestServer{})

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
	return grpcServer
}

type TestServer struct {
	pb.UnimplementedTestServiceServer
}

func (s *TestServer) Test(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	return &pb.TestResponse{Greetings: "Hello " + req.Name}, nil
}

func TestCore(t *testing.T) {

	addr := "localhost:8086"
	go makeGRPCServer(addr)

	// Make registry
	registry := NewRegistry()
	err := registry.Reload("./proto", "./test_service.proto")
	require.NoErrorf(t, err, "proto file should be loaded")

	// Get concrete service by name
	testService := registry.Service("TestService")
	require.NotNil(t, testService, "service should be founded")

	// Create grpc client
	err = testService.CreateClient(addr)
	require.NoErrorf(t, err, "service client should be created")

	// Call with structs
	req := &TestRequest{Name: "<Name>"}
	res := &TestResponse{}
	err = testService.Call("Test", req, res)
	require.NoErrorf(t, err, "error on gRPC call using structs")
	require.Equal(t, res.Greetings, "Hello "+req.Name, "incorrect server answer")

	// Call with json
	jsonReq := []byte(`{"Name": "<Name>"}`)
	jsonRes, err := testService.CallJson("Test", jsonReq)
	require.NoErrorf(t, err, "error on gRPC call using structs")

	// Checking response
	resMap := map[string]string{}
	err = json.Unmarshal(jsonRes, &resMap)
	require.NoErrorf(t, err, "error while unmarshal json response")
	require.Equal(t, resMap["Greetings"], "Hello "+req.Name, "incorrect server answer")
}
