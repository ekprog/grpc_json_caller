package grpc_caller

import (
	"log"
)

type TestRequest struct {
	Name string
}

type TestResponse struct {
	Greetings string
}

func example() {

	// Make registry
	registry := NewRegistry()
	err := registry.Reload("./proto", "./test_service.proto")
	if err != nil {
		log.Fatal(err)
	}

	// Discover services
	for i, serviceName := range registry.Services() {
		log.Printf("Service %d: %s\n", i+1, serviceName)
	}

	// Get concrete service by name
	testService := registry.Service("AuthService")
	if testService == nil {
		log.Fatal("Service does not exists")
	}

	// Discover methods
	for i, methodName := range testService.Methods() {
		log.Printf("Method %d: %s\n", i+1, methodName)
	}

	// Create grpc client
	err = testService.CreateClient("localhost:8086")
	if err != nil {
		log.Fatal(err)
	}

	// Call with structs
	req := &TestRequest{Name: "<Name>"}
	res := &TestResponse{}
	err = testService.Call("Test", req, res)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(res.Greetings)

	// Call with json
	jsonReq := []byte(`{"Name": "<My Name>"}`)
	jsonRes, err := testService.CallJson("Test", jsonReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", jsonRes)
}
