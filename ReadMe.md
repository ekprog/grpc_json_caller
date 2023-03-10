### About

Very simple wrapper for call gRPC methods at runtime with only loading .proto files.
\
\
Also suitable for routing HTTP REST requests to gRPC (for example, Gateway API).

- Dynamic routing
- Reflection in gRPC call
- Json Schema for calling gRPC method

### Installation

```shell
go get github.com/ekprog/grpc-json-caller
```

```go
import (
	grpcCaller "github.com/ekprog/grpc_json_caller"
)
```

### Examples
\
Your Proto file from gRPC server (or some files)

```protobuf
service TestService {
  rpc Test (TestRequest) returns (TestResponse) {}
}

message TestRequest {
  string Name = 1;
}
message TestResponse {
  string Greetings = 1;
}

```

\
Parsing proto and setup service with client

```go
// Make registry and parsing proto
// .proto - source dir with is root for imports from proto files
registry := NewRegistry()
err := registry.Reload("./proto", "./test_service.proto")
if err != nil {
    log.Fatal(err)
}

// Get Service with name equals in proto service name
service := registry.Service("TestService")
service.CreateClient("localhost:8086")

```
\
Call using structs
```go
type TestRequest struct {
	Name string
}

type TestResponse struct {
	Greetings string
}

func main() {
    // ... init steps
    
    var res TestResponse
    service.Call("Test", &TestRequest{Name: "<Name>"}, &res)
    log.Print(res.Greetings)
}
```

\
Call using JSON
```go
func main() {
    // ... init steps
    
    // Json schema should be equals to proto input message
    jsonReq := []byte(`{"Name": "<My Name>"}`)
    jsonRes, _ := service.CallJson("Test", jsonReq)
    log.Printf("%s", jsonRes)
}
```

### Documentation


```go
// Make registry
registry := NewRegistry()
registry.Reload("./proto", "./test_service.proto")

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
	
// Create default client
service.CreateClient("localhost:8086")

// Set custom client
conn, _ := grpc.Dial("localhost:8086", grpc.WithTransportCredentials(insecure.NewCredentials()))
service.SetClient(conn)

// Call JSON
jsonReq := []byte(`{"Name": "<My Name>"}`)
jsonRes, _ := service.CallJson("Test", jsonReq)

// Call JSON WithContext
jsonRes, _ := service.CallJsonWithContext(ctx, "Test", jsonReq)

// Call with object
req := &RequestObject{}
res := &ResponseObject{}
service.Call("Test", req, res)

// Call with object WithContext
service.CallWithContext("Test", req, res)
```


### Testing

For making tests you should generate server gRPC files

```shell
protoc -I ./proto \
--go_out ./proto \
--go_opt paths=source_relative \
--go-grpc_out ./proto \
--go-grpc_opt paths=source_relative \
./proto/test_service.proto
```