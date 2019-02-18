package main

import (
	"fmt"
	"log"
	"time"
	"golang.org/x/net/context"
	pb "grpc"
	"net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
)

const (
	port = ":8787"
)


func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGrpcServer(s, &server{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type server struct{}

// Typical RPC service, single request, single response
func (s *server) RpcService(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	log.Printf("RequestTimestamp: %v, Data: %v", in.Header.RequestTimestamp, in.Body.Data)
	in.Header.ResponseTimestamp = time.Now().Unix()
	response := &pb.Response{
		Header: in.Header,
		Body: &pb.Body{Data: fmt.Sprintf("Received data: %v", in.Body.Data)},
	}
	return response, nil
}

// single request, multiple responses
func (s *server) ServerSideStreamService(in *pb.Request, stream pb.Grpc_ServerSideStreamServiceServer) error {
	for i := 0; i < 5; i++ {
		in.Header.ResponseTimestamp = time.Now().Unix()
		response := &pb.Response{
			Header: in.Header,
			Body:   &pb.Body{Data: fmt.Sprintf("Received data: %v, dealing %v time(s)", in.Body.Data, i)},
		}
		if err := stream.Send(response); err != nil {
			return err
		}

		time.Sleep(time.Second)
	}

	return nil
}

// multiple requests, single response
func (s *server) ClientSideStreamService(stream pb.Grpc_ClientSideStreamServiceServer) error {
	index := 0
	var requestTimestamp int64
	for {
		in, err := stream.Recv()
		if index == 0 {
			requestTimestamp = in.Header.RequestTimestamp
		}
		// at this moment in == nil and err == io.EOF
		if err == io.EOF {
			response := &pb.Response{
				Header: &pb.Header{RequestTimestamp: requestTimestamp, ResponseTimestamp: time.Now().Unix()},
				Body: &pb.Body{Data: fmt.Sprintf("finished receiving data")},
			}
			log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", response.Header.RequestTimestamp, response.Header.ResponseTimestamp, response.Body.Data)
			return stream.SendAndClose(response)
		}
		if err != nil {
			return err
		}

		index++
	}
	return nil
}

// multiple requests, multiple responses
func (s *server) BidirectionalStreamService(stream pb.Grpc_BidirectionalStreamServiceServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", in.Header.RequestTimestamp, 0, in.Body.Data)

		in.Header.ResponseTimestamp = time.Now().Unix()
		response := &pb.Response{
			Header: in.Header,
			Body: &pb.Body{Data: fmt.Sprintf("server has received the request data")},
		}
		err = stream.Send(response)
		if err != nil {
			return err
		}
	}

	return nil
}