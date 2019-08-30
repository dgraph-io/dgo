package dgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"google.golang.org/grpc"
)

type School struct {
	Name       string `json:"name,omitempty"`
	DgraphType string `json:"dgraph.type,omitempty"`
}

type loc struct {
	Type       string    `json:"type,omitempty"`
	Coords     []float64 `json:"coordinates,omitempty"`
	DgraphType string    `json:"dgraph.type,omitempty"`
}

// If omitempty is not set, then edges with empty values (0 for int/float, "" for string, false
// for bool) would be created for values not specified explicitly.

type Person struct {
	Uid        string     `json:"uid,omitempty"`
	Name       string     `json:"name,omitempty"`
	Age        int        `json:"age,omitempty"`
	Dob        *time.Time `json:"dob,omitempty"`
	Married    bool       `json:"married,omitempty"`
	Raw        []byte     `json:"raw_bytes,omitempty"`
	Friends    []Person   `json:"friend,omitempty"`
	Location   loc        `json:"loc,omitempty"`
	School     []School   `json:"school,omitempty"`
	DgraphType string     `json:"dgraph.type,omitempty"`
}

func Example_setObject() {
	conn, err := grpc.Dial("127.0.0.1:9180", grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}
	defer conn.Close()

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)

	dob := time.Date(1980, 01, 01, 23, 0, 0, 0, time.UTC)
	// While setting an object if a struct has a Uid then its properties in the graph are updated
	// else a new node is created.
	// In the example below new nodes for Alice, Bob and Charlie and school are created (since they
	// dont have a Uid).
	p := Person{
		Uid:        "_:alice",
		Name:       "Alice",
		Age:        26,
		Married:    true,
		DgraphType: "Person",
		Location: loc{
			Type:       "Point",
			Coords:     []float64{1.1, 2},
			DgraphType: "Location",
		},
		Dob: &dob,
		Raw: []byte("raw_bytes"),
		Friends: []Person{{
			Name:       "Bob",
			Age:        24,
			DgraphType: "Person",
		}, {
			Name:       "Charlie",
			Age:        29,
			DgraphType: "Person",
		}},
		School: []School{{
			Name:       "Crown Public School",
			DgraphType: "Institution",
		}},
	}

	op := &api.Operation{}
	op.Schema = `
		name: string @index(exact) .
		age: int .
		married: bool .
		loc: geo .
		dob: datetime .

	    type Person {
			name: string
			age: int
			married: bool
			Friends: [Person]
		  }

		type Institution {
			name: string
		  }

		  type Location {
			type: string
			coords: float
		  }

	`

	ctx := context.Background()
	err = dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	mu := &api.Mutation{
		CommitNow: true,
	}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	assigned, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	// Assigned uids for nodes which were created would be returned in the assigned.Uids map.
	variables := map[string]string{"$id1": assigned.Uids["alice"]}
	q := `query Me($id1: string){
		me(func: uid($id1)) {
			name
			dob
			age
			loc
			raw_bytes
			married
			friend @filter(eq(name, "Bob")){
				name
				age
			}
			school {
				name
			}
		}
	}`

	resp, err := dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	type Root struct {
		Me []Person `json:"me"`
	}

	var r Root
	err = json.Unmarshal(resp.Json, &r)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("Me: %+v\n", r.Me)
	// R.Me would be same as the person that we set above.

	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"Alice","dob":"1980-01-01T23:00:00Z","age":26,"loc":{"type":"Point","coordinates":[1.1,2]},"raw_bytes":"cmF3X2J5dGVz","married":true,"friend":[{"name":"Bob","age":24}],"school":[{"name":"Crown Public School"}]}]}

}
