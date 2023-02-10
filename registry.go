package grpc_caller

import (
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type Registry struct {
	srcDir   string
	filename string
	services map[string]*Service
}

func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]*Service),
	}
}

func (r *Registry) Reload(srcDir, filename string) error {

	r.srcDir = srcDir
	r.filename = filename
	r.services = make(map[string]*Service)

	// File Registry
	tmpFile := path.Join(r.srcDir, r.filename+"-tmp.pb")
	cmd := exec.Command("protoc",
		"-I",
		path.Join(r.srcDir),
		"--include_imports",
		"--descriptor_set_out",
		tmpFile,
		path.Join(r.srcDir, r.filename))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	marshalledDescriptorSet, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		return err
	}

	descriptorSet := descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(marshalledDescriptorSet, &descriptorSet)
	if err != nil {
		return err
	}

	registry, err := protodesc.NewFiles(&descriptorSet)
	if err != nil {
		return err
	}

	_, nameOnly := filepath.Split(r.filename)
	desc, err := registry.FindFileByPath(nameOnly)
	if err != nil {
		return err
	}

	// Parse services
	services := desc.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		serviceObj, err := MakeService(service)
		if err != nil {
			return err
		}
		r.services[string(service.Name())] = serviceObj
	}
	return nil
}

func (r *Registry) ServiceExist(name string) bool {
	return r.Service(name) != nil
}

func (r *Registry) Service(name string) *Service {
	return r.services[name]
}

func (r *Registry) Services() []string {
	var list []string

	for _, method := range r.services {
		list = append(list, method.Name())
	}
	return list
}

func (r *Registry) Call(service, method string, in, out interface{}) error {
	serviceObj := r.services[service]
	if serviceObj == nil {
		return errors.New("service does not exist")
	}

	return serviceObj.Call(method, in, out)
}

func (r *Registry) CallJson(service, method string, jsonIn []byte) ([]byte, error) {
	serviceObj := r.services[service]
	if serviceObj == nil {
		return nil, errors.New("service does not exist")
	}
	return serviceObj.CallJson(method, jsonIn)
}

func (r *Registry) CallWithContext(ctx context.Context, service, method string, in, out interface{}) error {
	serviceObj := r.services[service]
	if serviceObj == nil {
		return errors.New("service does not exist")
	}
	return serviceObj.CallWithContext(ctx, method, in, out)
}

func (r *Registry) CallJsonWithContext(ctx context.Context, service, method string, jsonIn []byte) ([]byte, error) {
	serviceObj := r.services[service]
	if serviceObj == nil {
		return nil, errors.New("service does not exist")
	}
	return serviceObj.CallJsonWithContext(ctx, method, jsonIn)
}

type Service struct {
	conn    *grpc.ClientConn
	service protoreflect.ServiceDescriptor
	methods map[string]*Method
}

func MakeService(descriptor protoreflect.ServiceDescriptor) (*Service, error) {
	serviceObj := &Service{
		service: descriptor,
		methods: make(map[string]*Method),
	}

	// Methods
	methods := descriptor.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		methodObj, err := MakeMethod(serviceObj, method)
		if err != nil {
			return nil, err
		}
		serviceObj.methods[string(method.Name())] = methodObj
	}
	return serviceObj, nil
}

func (s *Service) Name() string {
	return string(s.service.Name())
}

func (s *Service) Methods() []string {
	var list []string

	for _, method := range s.methods {
		list = append(list, method.Name())
	}
	return list
}

func (s *Service) SetClient(client *grpc.ClientConn) {
	s.conn = client
}

func (s *Service) CreateClient(addr string) error {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	s.SetClient(conn)
	return nil
}

func (s *Service) CreateClientWithDial(addr string, option grpc.DialOption) error {
	conn, err := grpc.Dial(addr, option)
	if err != nil {
		return err
	}
	s.SetClient(conn)
	return nil
}

func (s *Service) Call(method string, in, out interface{}) error {
	methodObj := s.methods[method]
	if methodObj == nil {
		return errors.New("method does not exist")
	}
	return methodObj.Call(s.conn, in, out)
}

func (s *Service) CallJson(method string, jsonIn []byte) ([]byte, error) {
	methodObj := s.methods[method]
	if methodObj == nil {
		return nil, errors.New("method does not exist")
	}
	return methodObj.CallJson(s.conn, jsonIn)
}

func (s *Service) CallWithContext(ctx context.Context, method string, in, out interface{}) error {
	methodObj := s.methods[method]
	if methodObj == nil {
		return errors.New("method does not exist")
	}
	return methodObj.CallWithContext(ctx, s.conn, in, out)
}

func (s *Service) CallJsonWithContext(ctx context.Context, method string, jsonIn []byte) ([]byte, error) {
	methodObj := s.methods[method]
	if methodObj == nil {
		return nil, errors.New("method does not exist")
	}
	return methodObj.CallJsonWithContext(ctx, s.conn, jsonIn)
}

type Method struct {
	parent   *Service
	method   protoreflect.MethodDescriptor
	request  protoreflect.MessageDescriptor
	response protoreflect.MessageDescriptor
}

func MakeMethod(parent *Service, descriptor protoreflect.MethodDescriptor) (*Method, error) {

	methodObj := &Method{
		parent:   parent,
		method:   descriptor,
		request:  descriptor.Input(),
		response: descriptor.Output(),
	}

	return methodObj, nil
}

func (m *Method) Name() string {
	return string(m.method.Name())
}

func (m *Method) Call(conn *grpc.ClientConn, in interface{}, out interface{}) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return m.CallWithContext(ctx, conn, in, out)
}

func (m *Method) CallJson(conn *grpc.ClientConn, jsonInput []byte) ([]byte, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return m.CallJsonWithContext(ctx, conn, jsonInput)
}

func (m *Method) CallWithContext(ctx context.Context, conn *grpc.ClientConn, in interface{}, out interface{}) error {

	jsonInput, err := json.Marshal(in)
	if err != nil {
		return err
	}

	jsonOutput, err := m.CallJsonWithContext(ctx, conn, jsonInput)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonOutput, out)
	if err != nil {
		return err
	}

	return nil
}

func (m *Method) CallJsonWithContext(ctx context.Context, conn *grpc.ClientConn, jsonInput []byte) ([]byte, error) {

	requestObj := dynamicpb.NewMessage(m.request)
	responseObj := dynamicpb.NewMessage(m.response)

	err := protojson.Unmarshal(jsonInput, requestObj)
	if err != nil {
		return nil, err
	}

	caller := "/" + string(m.parent.service.FullName()) + "/" + string(m.method.Name())
	err = conn.Invoke(ctx, caller, requestObj, responseObj)
	if err != nil {
		return nil, err
	}

	jsonOutput, err := protojson.Marshal(responseObj)
	if err != nil {
		return nil, err
	}
	return jsonOutput, nil
}
