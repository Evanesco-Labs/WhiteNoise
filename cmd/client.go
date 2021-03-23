package main

import "google.golang.org/grpc"

func NewClient(port string) (RpcClient, error) {
	conn, err := grpc.Dial("localhost:"+port, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := NewRpcClient(conn)
	return client, nil
}
