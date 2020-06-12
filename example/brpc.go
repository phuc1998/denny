package main

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/zipkin"
	"github.com/whatvn/denny"
	pb "github.com/whatvn/denny/example/protobuf"
	"github.com/whatvn/denny/middleware/http"
	"io"
)

// grpc
type Hello struct{}

func (s *Hello) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	var (
		logger = denny.GetLogger(ctx)
	)
	response := &pb.HelloResponse{
		Reply: "hi",
	}

	logger.WithField("response", response)
	return response, nil
}

func (s *Hello) SayHelloAnonymous(ctx context.Context, in *empty.Empty) (*pb.HelloResponse, error) {

	var (
		logger = denny.GetLogger(ctx)
	)

	span, ctx := opentracing.StartSpanFromContext(ctx, "sayHello")

	logger.Infof("span %v", span.Tracer() != nil)
	defer span.Finish()
	response := &pb.HelloResponse{
		Reply: "hi",
	}

	logger.WithField("response", response)

	return response, nil
}

type TestController struct {
	denny.Controller
}

func (t *TestController) Handle(ctx *denny.Context) {
	ctx.JSON(200, &pb.HelloResponse{
		Reply: "hi",
	})
}

func newReporterUDP(jaegerAddr string, port int, packetLength int) jaeger.Transport {
	hostString := fmt.Sprintf("%s:%d", jaegerAddr, port)
	transport, err := jaeger.NewUDPTransport(hostString, packetLength)
	if err != nil {
		panic(err)
	}
	return transport
}
func initTracerUDP(jaegerAddr string, port int, packetLength int, serviceName string) (opentracing.Tracer, io.Closer) {
	var (
		propagator = zipkin.NewZipkinB3HTTPHeaderPropagator()
	)

	return jaeger.NewTracer(
		serviceName,
		jaeger.NewConstSampler(true),
		jaeger.NewRemoteReporter(newReporterUDP(jaegerAddr, port, packetLength)),
		jaeger.TracerOptions.Injector(opentracing.HTTPHeaders, propagator),
		jaeger.TracerOptions.Extractor(opentracing.HTTPHeaders, propagator),
		jaeger.TracerOptions.ZipkinSharedRPCSpan(true),
	)
}

func main() {
	tracer, _ := initTracerUDP(
		"127.0.0.1",
		6831,
		65000,
		"brpc.culi.ngu.nhu.heo",
	)

	tracer.StartSpan("hello")
	opentracing.SetGlobalTracer(tracer)

	server := denny.NewServer(true)
	server.Use(http.Logger())
	group := server.NewGroup("/hi")
	group.Controller("/hi", denny.HttpPost, new(TestController))

	// setup grpc server

	grpcServer := denny.NewGrpcServer()
	pb.RegisterHelloServiceServer(grpcServer, new(Hello))
	server.WithGrpcServer(grpcServer)
	//

	//// then http
	authorized := server.NewGroup("/")
	authorized.BrpcController(&Hello{})

	// start server in dual mode
	server.GraceFulStart()
}