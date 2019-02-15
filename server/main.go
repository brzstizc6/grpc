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

func (s *server) RpcService(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	log.Printf("RequestTimestamp: %v, Data: %v", in.Header.RequestTimestamp, in.Body.Data)
	in.Header.ResponseTimestamp = time.Now().Unix()
	response := &pb.Response{
		Header: in.Header,
		Body: &pb.Body{Data: fmt.Sprintf("Received data: %v", in.Body.Data)},
	}
	return response, nil
}

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
				Body: &pb.Body{Data: fmt.Sprintf("Totally requested %v time(s)", index)},
			}
			return stream.SendAndClose(response)
		}
		if err != nil {
			return err
		}

		index++
	}
	return nil
}
