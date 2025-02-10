package api

import (
	grpc "google.golang.org/grpc"
)

func GetConn(c DgraphClient) grpc.ClientConnInterface {
	return c.(*dgraphClient).cc
}
