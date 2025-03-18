package dgo_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dgraph-io/dgo/v240"
)

func Test_connectLocal(t *testing.T) {
	dg, err := dgo.Connect("dgraph://127.0.0.1:9080")
	if err != nil {
		t.Fatalf("Failed to connect to local Dgraph: %v", err)
	}
	defer dg.Close()
	// Query to schema to check the connection
	resp, err := dg.NewReadOnlyTxn().BestEffort().Query(context.Background(), "schema {}")
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}
	fmt.Println(string(resp.Json))
}

func Test_connectLocalACL(t *testing.T) {
	dg, err := dgo.Connect("dgraph://groot:password@127.0.0.1:9080")
	if err != nil {
		t.Fatalf("Failed to connect to local Dgraph: %v", err)
	}
	defer dg.Close()
	// Query to schema to check the connection
	resp, err := dg.NewReadOnlyTxn().BestEffort().Query(context.Background(), "schema {}")
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}
	fmt.Println(string(resp.Json))
}

func Test_connectLocalACLAsUser(t *testing.T) {
	dg, err := dgo.Connect("dgraph://alice:supersecret@127.0.0.1:9080")
	if err != nil {
		t.Fatalf("Failed to connect to local Dgraph: %v", err)
	}
	defer dg.Close()
	// Query to schema to check the connection
	resp, err := dg.NewReadOnlyTxn().BestEffort().Query(context.Background(), "schema {}")
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}
	fmt.Println(string(resp.Json))
}

func Test_connectCloud(t *testing.T) {
	// https://throbbing-field-510005.us-west-2.aws.cloud.dgraph.io/graphql
	// throbbing-field-510005.grpc.us-west-2.aws.cloud.dgraph.io:443
	host := "throbbing-field-510005.grpc.us-west-2.aws.cloud.dgraph.io:443"
	apiKey := "NWEzM2VkNmI3MTM3YjIxN2ExYjQxYmJkODFlYWJhMWI="
	connStr := fmt.Sprintf("dgraph://%s?api-key=%s", host, apiKey)
	dg, err := dgo.Connect(connStr)
	if err != nil {
		t.Fatalf("Failed to connect to local Dgraph: %v", err)
	}
	defer dg.Close()

	// Check the connection
	query := `
	{
		me(func: uid(1)) {
			uid
		}
	}`
	resp, err := dg.NewReadOnlyTxn().BestEffort().Query(context.Background(), query)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}
	fmt.Println(string(resp.Json))
}
