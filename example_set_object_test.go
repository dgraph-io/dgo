package dgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v200/protos/api"
)

type School struct {
	Name  string   `json:"name,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`
}

type loc struct {
	Type   string    `json:"type,omitempty"`
	Coords []float64 `json:"coordinates,omitempty"`
}

// If omitempty is not set, then edges with empty values (0 for int/float, "" for string, false
// for bool) would be created for values not specified explicitly.

type Person struct {
	Uid      string     `json:"uid,omitempty"`
	Name     string     `json:"name,omitempty"`
	Age      int        `json:"age,omitempty"`
	Dob      *time.Time `json:"dob,omitempty"`
	Married  bool       `json:"married,omitempty"`
	Raw      []byte     `json:"raw_bytes,omitempty"`
	Friends  []Person   `json:"friend,omitempty"`
	Location loc        `json:"loc,omitempty"`
	School   []School   `json:"school,omitempty"`
	DType    []string   `json:"dgraph.type,omitempty"`
}

func Example_setObject() {
	dg, cancel := getDgraphClient()
	defer cancel()

	dob := time.Date(1980, 01, 01, 23, 0, 0, 0, time.UTC)
	// While setting an object if a struct has a Uid then its properties in the graph are updated
	// else a new node is created.
	// In the example below new nodes for Alice, Bob and Charlie and school are created (since they
	// don't have a Uid).
	p := Person{
		Uid:     "_:alice",
		Name:    "Alice",
		Age:     26,
		Married: true,
		DType:   []string{"Person"},
		Location: loc{
			Type:   "Point",
			Coords: []float64{1.1, 2},
		},
		Dob: &dob,
		Raw: []byte("raw_bytes"),
		Friends: []Person{{
			Name:  "Bob",
			Age:   24,
			DType: []string{"Person"},
		}, {
			Name:  "Charlie",
			Age:   29,
			DType: []string{"Person"},
		}},
		School: []School{{
			Name:  "Crown Public School",
			DType: []string{"Institution"},
		}},
	}

	op := &api.Operation{}
	op.Schema = `
		name: string @index(exact) .
		age: int .
		married: bool .
		loc: geo .
		dob: datetime .
		Friend: [uid] .
		type: string .
		coords: float .

		type Person {
			name: string
			age: int
			married: bool
			Friend: [Person]
			loc: Loc
		}

		type Institution {
			name: string
		}

		type Loc {
			type: string
			coords: float
		}
	`

	ctx := context.Background()
	if err := dg.Alter(ctx, op); err != nil {
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
	response, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	// Assigned uids for nodes which were created would be returned in the response.Uids map.
	variables := map[string]string{"$id1": response.Uids["alice"]}
	q := `query Me($id1: string){
		me(func: uid($id1)) {
			name
			dob
			age
			loc
			raw_bytes
			married
			dgraph.type
			friend @filter(eq(name, "Bob")){
				name
				age
				dgraph.type
			}
			school {
				name
				dgraph.type
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

	out, _ := json.MarshalIndent(r, "", "\t")
	fmt.Printf("%s\n", out)
	// Output: {
	// 	"me": [
	// 		{
	// 			"name": "Alice",
	// 			"age": 26,
	// 			"dob": "1980-01-01T23:00:00Z",
	// 			"married": true,
	// 			"raw_bytes": "cmF3X2J5dGVz",
	// 			"friend": [
	// 				{
	// 					"name": "Bob",
	// 					"age": 24,
	// 					"loc": {},
	// 					"dgraph.type": [
	// 						"Person"
	// 					]
	// 				}
	// 			],
	// 			"loc": {
	// 				"type": "Point",
	// 				"coordinates": [
	// 					1.1,
	// 					2
	// 				]
	// 			},
	// 			"school": [
	// 				{
	// 					"name": "Crown Public School",
	// 					"dgraph.type": [
	// 						"Institution"
	// 					]
	// 				}
	// 			],
	// 			"dgraph.type": [
	// 				"Person"
	// 			]
	// 		}
	// 	]
	// }
}
