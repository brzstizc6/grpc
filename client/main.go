package main

import (
	pb "grpc"
	"golang.org/x/net/context"
	"time"
	"log"
	"os"
	"google.golang.org/grpc"
	"io"
	"fmt"
)

const (
	address = "localhost:8787"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewGrpcClient(conn)

	mode := "rpc"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "rpc":
		rpc(client)
	case "serverSide":
		serverSideStream(client)
	case "clientSide":
		clientSideStream(client)
	case "bidirectional":
		BidirectionalStream(client)
	default:
		rpc(client)
	}
}

func rpc(client pb.GrpcClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	request := pb.Request{
		Header: &pb.Header{RequestTimestamp: time.Now().Unix()},
		Body: &pb.Body{Data: "hello world"},
	}
	r, err := client.RpcService(ctx, &request)
	if err != nil {
		log.Fatalf("Failure: %v", err)
		return err
	}

	log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", r.Header.RequestTimestamp, r.Header.ResponseTimestamp, r.Body.Data)

	return nil
}

func serverSideStream(client pb.GrpcClient) error {
	request := &pb.Request{
		Header: &pb.Header{RequestTimestamp: time.Now().Unix()},
		Body: &pb.Body{Data: "starting request"},
	}
	stream, err := client.ServerSideStreamService(context.Background(), request)
	if err != nil {
		log.Fatalf("Failure: %v", err)
		return err
	}

	log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", request.Header.RequestTimestamp, 0, request.Body.Data)

	for {
		r, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failure: %v", err)
			return err
		}

		log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", r.Header.RequestTimestamp, r.Header.ResponseTimestamp, r.Body.Data)
	}

	return nil
}

func clientSideStream(client pb.GrpcClient) error {
	stream, err := client.ClientSideStreamService(context.Background())
	if err != nil {
		log.Fatalf("Failure: %v", err)
		return err
	}

	requestTimestamp := time.Now().Unix()
	for i := 0; i < 5 ; i++ {
		request := &pb.Request{
			Header: &pb.Header{RequestTimestamp: requestTimestamp},
			Body: &pb.Body{Data: fmt.Sprintf("request %v time(s)", i)},
		}
		err := stream.Send(request)
		if err != nil {
			log.Fatalf("Failure: %v", err)
			return err
		}

		log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", request.Header.RequestTimestamp, 0, request.Body.Data)

		time.Sleep(time.Second)
	}

	// CloseAndRecv() method will close the stream, sending err == io.EOF to the server side
	r, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Failure: %v", err)
		return err
	}

	log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", r.Header.RequestTimestamp, r.Header.ResponseTimestamp, r.Body.Data)

	return nil
}

func BidirectionalStream(client pb.GrpcClient) error {
	stream, err := client.BidirectionalStreamService(context.Background())
	if err != nil {
		log.Fatalf("Failure: %v", err)
		return err
	}

	for i := 0; i < 5; i++ {
		request := &pb.Request{
			Header: &pb.Header{RequestTimestamp: time.Now().Unix()},
			Body: &pb.Body{Data: fmt.Sprintf("client has sent request data")},
		}
		err = stream.Send(request)
		if err != nil {
			log.Fatalf("Failure: %v", err)
			return err
		}

		time.Sleep(time.Second)

		r, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failure: %v", err)
			return err
		}
		log.Printf("RequestTimestamp: %v, ResponseTimestamp: %v, Message: %v", r.Header.RequestTimestamp, r.Header.ResponseTimestamp, r.Body.Data)
	}

	// close the stream
	stream.CloseSend()

	return nil
}