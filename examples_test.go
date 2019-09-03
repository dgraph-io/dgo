/*
 * Copyright (C) 2017 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"google.golang.org/grpc"
)

type CancelFunc func()

func getDgraphClient() (*dgo.Dgraph, CancelFunc) {
	conn, err := grpc.Dial("127.0.0.1:9180", grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)
	ctx := context.Background()

	// Perform login call. If the Dgraph cluster does not have ACL and
	// enterprise features enabled, this call should be skipped.
	for {
		// Keep retrying until we succeed or receive a non-retriable error.
		err = dg.Login(ctx, "groot", "password")
		if err == nil || !strings.Contains(err.Error(), "Please retry") {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("While trying to login %v", err.Error())
	}

	return dg, func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error while closing connection:%v", err)
		}
	}
}

func ExampleDgraph_Alter_dropAll() {
	dg, cancel := getDgraphClient()
	defer cancel()
	op := api.Operation{DropAll: true}
	ctx := context.Background()
	if err := dg.Alter(ctx, &op); err != nil {
		log.Fatal(err)
	}
	// Output:
}

func ExampleTxn_Query_variables() {
	dg, cancel := getDgraphClient()
	defer cancel()
	type Person struct {
		Uid   string `json:"uid,omitempty"`
		Name  string `json:"name,omitempty"`
		DType string `json:"dgraph.type,omitempty"`
	}

	op := &api.Operation{}
	op.Schema = `
		name: string @index(exact) .

		type Person {
			name: string
		  }

	`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	p := Person{
		Name:  "Alice",
		DType: "Person",
	}

	mu := &api.Mutation{
		CommitNow: true,
	}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	_, err = dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	variables := make(map[string]string)
	variables["$a"] = "Alice"
	q := `query Alice($a: string){
		me(func: eq(name, $a)) {
			name
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

	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"Alice"}]}
}

func ExampleTxn_Mutate() {
	type School struct {
		Name  string `json:"name,omitempty"`
		DType string `json:"dgraph.type,omitempty"`
	}

	type loc struct {
		Type   string    `json:"type,omitempty"`
		Coords []float64 `json:"coordinates,omitempty"`
	}

	// If omitempty is not set, then edges with empty values (0 for int/float, "" for string, false
	// for bool) would be created for values not specified explicitly.

	type Person struct {
		Uid      string   `json:"uid,omitempty"`
		Name     string   `json:"name,omitempty"`
		Age      int      `json:"age,omitempty"`
		Married  bool     `json:"married,omitempty"`
		Raw      []byte   `json:"raw_bytes,omitempty"`
		Friends  []Person `json:"friend,omitempty"`
		Location loc      `json:"loc,omitempty"`
		School   []School `json:"school,omitempty"`
		DType    string   `json:"dgraph.type,omitempty"`
	}

	dg, cancel := getDgraphClient()
	defer cancel()
	// While setting an object if a struct has a Uid then its properties in the graph are updated
	// else a new node is created.
	// In the example below new nodes for Alice, Bob and Charlie and school are created (since they
	// don't have a Uid).
	p := Person{
		Uid:     "_:alice",
		Name:    "Alice",
		Age:     26,
		Married: true,
		DType:   "Person",
		Location: loc{
			Type:   "Point",
			Coords: []float64{1.1, 2},
		},
		Raw: []byte("raw_bytes"),
		Friends: []Person{{
			Name:  "Bob",
			Age:   24,
			DType: "Person",
		}, {
			Name:  "Charlie",
			Age:   29,
			DType: "Person",
		}},
		School: []School{{
			Name:  "Crown Public School",
			DType: "Institution",
		}},
	}

	op := &api.Operation{}
	op.Schema = `
		age: int .
		married: bool .

	    type Person {
			name: string
			age: int
			married: bool
			Friends:[Person]
		  }

		type Location {
			type: string
			coords: float
		  }

		type Institution {
			name: string
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
	assigned, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	// Assigned uids for nodes which were created would be returned in the assigned.Uids map.
	puid := assigned.Uids["alice"]
	const q = `query Me($id: string){
		me(func: uid($id)) {
			name
			age
			loc
			raw_bytes
			married
			friend @filter(eq(name, "Bob")) {
				name
				age
			}
			school {
				name
			}
		}
	}`

	variables := make(map[string]string)
	variables["$id"] = puid
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

	// R.Me would be same as the person that we set above.
	// fmt.Printf("Me: %+v\n", r.Me)

	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"Alice","age":26,"loc":{"type":"Point","coordinates":[1.1,2]},"raw_bytes":"cmF3X2J5dGVz","married":true,"friend":[{"name":"Bob","age":24}],"school":[{"name":"Crown Public School"}]}]}

}

func ExampleTxn_Mutate_bytes() {
	dg, cancel := getDgraphClient()
	defer cancel()
	type Person struct {
		Uid   string `json:"uid,omitempty"`
		Name  string `json:"name,omitempty"`
		Bytes []byte `json:"bytes,omitempty"`
		DType string `json:"dgraph.type,omitempty"`
	}

	op := &api.Operation{}
	op.Schema = `
		name: string @index(exact) .

	    type Person {
			name: string
			bytes: string
		  }

	`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	p := Person{
		Name:  "Alice-new",
		DType: "Person",
		Bytes: []byte("raw_bytes"),
	}

	mu := &api.Mutation{
		CommitNow: true,
	}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	_, err = dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	q := `{
	q(func: eq(name, "Alice-new")) {
		name
		bytes
	}
}`

	resp, err := dg.NewTxn().Query(ctx, q)
	if err != nil {
		log.Fatal(err)
	}

	type Root struct {
		Me []Person `json:"q"`
	}

	var r Root
	err = json.Unmarshal(resp.Json, &r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Me: %+v\n", r.Me)

	// Output: Me: [{Uid: Name:Alice-new Bytes:[114 97 119 95 98 121 116 101 115]}]
}

func ExampleTxn_Query_unmarshal() {
	type School struct {
		Name  string `json:"name,omitempty"`
		DType string `json:"dgraph.type,omitempty"`
	}

	type Person struct {
		Uid     string   `json:"uid,omitempty"`
		Name    string   `json:"name,omitempty"`
		Age     int      `json:"age,omitempty"`
		Married bool     `json:"married,omitempty"`
		Raw     []byte   `json:"raw_bytes,omitempty"`
		Friends []Person `json:"friend,omitempty"`
		School  []School `json:"school,omitempty"`
		DType   string   `json:"dgraph.type,omitempty"`
	}

	dg, cancel := getDgraphClient()
	defer cancel()
	op := &api.Operation{}
	op.Schema = `
		age: int .
		married: bool .

	    type Person {
			name: string
			age: int
			married: bool
			Friends: [Person]
		  }

		type Institution {
			name: string
		  }

	`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	p := Person{
		Uid:   "_:bob",
		Name:  "Bob",
		Age:   24,
		DType: "Person",
	}

	txn := dg.NewTxn()
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu := &api.Mutation{
		CommitNow: true,
		SetJson:   pb,
	}
	assigned, err := txn.Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}
	bob := assigned.Uids["bob"]

	// While setting an object if a struct has a Uid then its properties in the graph are updated
	// else a new node is created.
	// In the example below new nodes for Alice and Charlie and school are created (since they dont
	// have a Uid).  Alice is also connected via the friend edge to an existing node Bob.
	p = Person{
		Uid:     "_:alice",
		Name:    "Alice",
		Age:     26,
		Married: true,
		DType:   "Person",
		Raw:     []byte("raw_bytes"),
		Friends: []Person{{
			Uid: bob,
		}, {
			Name:  "Charlie",
			Age:   29,
			DType: "Person",
		}},
		School: []School{{
			Name:  "Crown Public School",
			DType: "Institution",
		}},
	}

	txn = dg.NewTxn()
	mu = &api.Mutation{}
	pb, err = json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	mu.CommitNow = true
	assigned, err = txn.Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	// Assigned uids for nodes which were created would be returned in the assigned.Uids map.
	puid := assigned.Uids["alice"]
	variables := make(map[string]string)
	variables["$id"] = puid
	const q = `query Me($id: string){
		me(func: uid($id)) {
			name
			age
			loc
			raw_bytes
			married
			friend @filter(eq(name, "Bob")) {
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

	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"Alice","age":26,"raw_bytes":"cmF3X2J5dGVz","married":true,"friend":[{"name":"Bob","age":24}],"school":[{"name":"Crown Public School"}]}]}
}

func ExampleTxn_Query_besteffort() {
	dg, cancel := getDgraphClient()
	defer cancel()

	// NOTE: Best effort only works with read-only queries.
	txn := dg.NewReadOnlyTxn().BestEffort()
	resp, err := txn.Query(context.Background(), `{ q(func: uid(0x1)) { uid } }`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(resp.Json))
	// Output: {"q":[{"uid":"0x1"}]}
}

func ExampleTxn_Mutate_facets() {
	dg, cancel := getDgraphClient()
	defer cancel()
	// Doing a dropAll isn't required by the user. We do it here so that we can verify that the
	// example runs as expected.
	op := api.Operation{DropAll: true}
	ctx := context.Background()
	if err := dg.Alter(ctx, &op); err != nil {
		log.Fatal(err)
	}

	op = api.Operation{}
	op.Schema = `
		name: string @index(exact) .

	    type Person {
			name: string
			age: int
			married: bool
			NameOrigin: string
			Since: string
			Family: string
			Age: int
			Close: bool
			Friends: [Person]
		  }

		type Institution {
			name: string
			Since: string
		  }

	`

	err := dg.Alter(ctx, &op)
	if err != nil {
		log.Fatal(err)
	}

	// This example shows example for SetObject using facets.
	type School struct {
		Name  string    `json:"name,omitempty"`
		Since time.Time `json:"school|since,omitempty"`
		DType string    `json:"dgraph.type,omitempty"`
	}

	type Person struct {
		Uid        string   `json:"uid,omitempty"`
		Name       string   `json:"name,omitempty"`
		NameOrigin string   `json:"name|origin,omitempty"`
		Friends    []Person `json:"friend,omitempty"`

		// These are facets on the friend edge.
		Since  time.Time `json:"friend|since,omitempty"`
		Family string    `json:"friend|family,omitempty"`
		Age    float64   `json:"friend|age,omitempty"`
		Close  bool      `json:"friend|close,omitempty"`

		School []School `json:"school,omitempty"`
		DType  string   `json:"dgraph.type,omitempty"`
	}

	ti := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	p := Person{
		Uid:        "_:alice",
		Name:       "Alice",
		NameOrigin: "Indonesia",
		DType:      "Person",
		Friends: []Person{
			Person{
				Name:   "Bob",
				Since:  ti,
				Family: "yes",
				Age:    13,
				Close:  true,
				DType:  "Person",
			},
			Person{
				Name:   "Charlie",
				Family: "maybe",
				Age:    16,
				DType:  "Person",
			},
		},
		School: []School{School{
			Name:  "Wellington School",
			Since: ti,
			DType: "Institution",
		}},
	}

	mu := &api.Mutation{}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	mu.CommitNow = true
	assigned, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	auid := assigned.Uids["alice"]
	variables := make(map[string]string)
	variables["$id"] = auid

	const q = `query Me($id: string){
        me(func: uid($id)) {
            name @facets
			friend @filter(eq(name, "Bob")) @facets {
				name
				dgraph.type
            }
            school @facets {
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

	fmt.Printf("Me: %+v\n", r.Me)
	// Output: Me: [{Uid: Name:Alice NameOrigin:Indonesia Friends:[{Uid: Name:Bob NameOrigin: Friends:[] Since:2009-11-10 23:00:00 +0000 UTC Family:yes Age:13 Close:true School:[]}] Since:0001-01-01 00:00:00 +0000 UTC Family: Age:0 Close:false School:[{Name:Wellington School Since:2009-11-10 23:00:00 +0000 UTC}]}]
}

func ExampleTxn_Mutate_list() {
	dg, cancel := getDgraphClient()
	defer cancel() // This example shows example for SetObject for predicates with list type.
	type Person struct {
		Uid         string   `json:"uid"`
		Address     []string `json:"address"`
		PhoneNumber []int64  `json:"phone_number"`
		DType       string   `json:"dgraph.type,omitempty"`
	}

	p := Person{
		Uid:         "_:person",
		Address:     []string{"Redfern", "Riley Street"},
		PhoneNumber: []int64{9876, 123},
		DType:       "Person",
	}

	op := &api.Operation{}
	op.Schema = `
		address: [string] .
		phone_number: [int] .

		type Person {
			Address: [string]
			phone_number: [int]
		  }

	`
	ctx := context.Background()
	err := dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	mu := &api.Mutation{}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	mu.CommitNow = true
	assigned, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	variables := map[string]string{"$id": assigned.Uids["person"]}
	const q = `
	query Me($id: string){
		me(func: uid($id)) {
			address
			phone_number
			dgraph.type
		}
	}
	`

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

	// List items aren't guaranteed to be in the same order.
	fmt.Println(string(resp.Json))
	// {"me":[{"address":["Riley Street","Redfern"],"phone_number":[9876,123]}]}

}

func ExampleDeleteEdges() {
	dg, cancel := getDgraphClient()
	defer cancel()
	op := &api.Operation{}
	op.Schema = `
			age: int .
			married: bool .
			name: string @lang .
			location: string .

			type Person {
				name: string
				age: int
				married: bool
				Friends: [Person]
			  }
		
			type Institution {
				name: string
			  }

		`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	type School struct {
		Uid   string `json:"uid"`
		Name  string `json:"name@en,omitempty"`
		DType string `json:"dgraph.type,omitempty"`
	}

	type Person struct {
		Uid      string    `json:"uid,omitempty"`
		Name     string    `json:"name,omitempty"`
		Age      int       `json:"age,omitempty"`
		Married  bool      `json:"married,omitempty"`
		Friends  []Person  `json:"friends,omitempty"`
		Location string    `json:"location,omitempty"`
		Schools  []*School `json:"schools,omitempty"`
		DType    string    `json:"dgraph.type,omitempty"`
	}

	// Lets add some data first.
	p := Person{
		Uid:      "_:alice",
		Name:     "Alice",
		Age:      26,
		Married:  true,
		DType:    "Person",
		Location: "Riley Street",
		Friends: []Person{{
			Name:  "Bob",
			Age:   24,
			DType: "Person",
		}, {
			Name:  "Charlie",
			Age:   29,
			DType: "Person",
		}},
		Schools: []*School{&School{
			Name:  "Crown Public School",
			DType: "Institution",
		}},
	}

	mu := &api.Mutation{}
	pb, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	mu.SetJson = pb
	mu.CommitNow = true
	assigned, err := dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	alice := assigned.Uids["alice"]

	variables := make(map[string]string)
	variables["$alice"] = alice
	const q = `query Me($alice: string){
		me(func: uid($alice)) {
			name
			age
			location
			married
			dgraph.type
			friends {
				name
				age
				dgraph.type
			}
			schools {
				name@en
				dgraph.type
			}
		}
	}`

	resp, err := dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	// Now lets delete the friend and location edge from Alice
	mu = &api.Mutation{}
	dgo.DeleteEdges(mu, alice, "friends", "location")

	mu.CommitNow = true
	_, err = dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	type Root struct {
		Me []Person `json:"me"`
	}

	var r Root
	err = json.Unmarshal(resp.Json, &r)
	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"Alice","age":26,"married":true,"schools":[{"name@en":"Crown Public School"}]}]}
}

func ExampleTxn_Mutate_deleteNode() {
	dg, cancel := getDgraphClient()
	defer cancel()
	// In this test we check S * * deletion.
	type Person struct {
		Uid     string    `json:"uid,omitempty"`
		Name    string    `json:"name,omitempty"`
		Age     int       `json:"age,omitempty"`
		Married bool      `json:"married,omitempty"`
		Friends []*Person `json:"friend,omitempty"`
		DType   string    `json:"dgraph.type,omitempty"`
	}

	p := Person{
		Uid:     "_:alice",
		Name:    "Alice",
		Age:     26,
		Married: true,
		DType:   "Person",
		Friends: []*Person{&Person{
			Uid:   "_:bob",
			Name:  "Bob",
			Age:   24,
			DType: "Person",
		}, &Person{
			Uid:   "_:charlie",
			Name:  "Charlie",
			Age:   29,
			DType: "Person",
		}},
	}

	op := &api.Operation{}
	op.Schema = `
		age: int .
		married: bool .
        type Person {
          name: string
          age: int
          married: bool
          friend: [Person]
        }
	`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
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

	alice := assigned.Uids["alice"]
	bob := assigned.Uids["bob"]
	charlie := assigned.Uids["charlie"]

	variables := make(map[string]string)
	variables["$alice"] = alice
	variables["$bob"] = bob
	variables["$charlie"] = charlie
	const q = `query Me($alice: string, $bob: string, $charlie: string){
		me(func: uid($alice)) {
			name
			age
			married
			dgraph.type
			friend {
				uid
				name
				age
				dgraph.type
			}
		}

		me2(func: uid($bob)) {
			name
			age
			dgraph.type
		}

		me3(func: uid($charlie)) {
			name
			age
			dgraph.type
		}
	}`

	resp, err := dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	type Root struct {
		Me  []Person `json:"me"`
		Me2 []Person `json:"me2"`
		Me3 []Person `json:"me3"`
	}

	var r Root
	err = json.Unmarshal(resp.Json, &r)

	// Now lets try to delete Alice. This won't delete Bob and Charlie but just remove the
	// connection between Alice and them.

	// The JSON for deleting a node should be of the form {"uid": "0x123"}. If you wanted to
	// delete multiple nodes you could supply an array of objects like [{"uid": "0x321"}, {"uid":
	// "0x123"}] to DeleteJson.

	d := map[string]string{"uid": alice}
	pb, err = json.Marshal(d)
	if err != nil {
		log.Fatal(err)
	}

	mu = &api.Mutation{
		CommitNow:  true,
		DeleteJson: pb,
	}

	_, err = dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(resp.Json, &r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Resp after deleting node: %+v\n", string(resp.Json))
	// Output: Resp after deleting node: {"me":[],"me2":[{"name":"Bob","age":24}],"me3":[{"name":"Charlie","age":29}]}
}

func ExampleTxn_Mutate_deletePredicate() {
	dg, cancel := getDgraphClient()
	defer cancel()
	type Person struct {
		Uid     string   `json:"uid,omitempty"`
		Name    string   `json:"name,omitempty"`
		Age     int      `json:"age,omitempty"`
		Married bool     `json:"married,omitempty"`
		Friends []Person `json:"friend,omitempty"`
		DType   string   `json:"dgraph.type,omitempty"`
	}

	p := Person{
		Uid:     "_:alice",
		Name:    "Alice",
		Age:     26,
		Married: true,
		DType:   "Person",
		Friends: []Person{Person{
			Name:  "Bob",
			Age:   24,
			DType: "Person",
		}, Person{
			Name:  "Charlie",
			Age:   29,
			DType: "Person",
		}},
	}

	op := &api.Operation{}
	op.Schema = `
		age: int .
		married: bool .

	    type Person {
			name: string
			age: int
			married: bool
			Friends: [Person]
		  }

	`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
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

	alice := assigned.Uids["alice"]

	variables := make(map[string]string)
	variables["$id"] = alice
	const q = `query Me($id: string){
		me(func: uid($id)) {
			name
			age
			married
			dgraph.type
			friend {
				uid
				name
				age
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

	op = &api.Operation{DropAttr: "friend"}
	err = dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	op = &api.Operation{DropAttr: "married"}
	err = dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	// Also lets run the query again to verify that predicate data was deleted.
	resp, err = dg.NewTxn().QueryWithVars(ctx, q, variables)
	if err != nil {
		log.Fatal(err)
	}

	r = Root{}
	err = json.Unmarshal(resp.Json, &r)
	if err != nil {
		log.Fatal(err)
	}

	// Alice should have no friends and only two attributes now.
	fmt.Printf("Response after deletion: %+v\n", r)
	// Output: Response after deletion: {Me:[{Uid: Name:Alice Age:26 Married:false Friends:[]}]}
}

func ExampleTxn_Discard() {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx, toCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer toCancel()
	err := dg.Alter(ctx, &api.Operation{
		DropAll: true,
	})
	if err != nil {
		log.Fatal("The drop all operation should have succeeded")
	}

	err = dg.Alter(ctx, &api.Operation{
		Schema: `name: string @index(exact) .`,
	})
	if err != nil {
		log.Fatal("The alter should have succeeded")
	}

	txn := dg.NewTxn()
	_, err = txn.Mutate(ctx, &api.Mutation{
		SetNquads: []byte(`_:a <name> "Alice" .`),
	})
	if err != nil {
		log.Fatal("The mutation should have succeeded")
	}
	txn.Discard(ctx)

	// now query the cluster and make sure that the data has made no effect
	queryTxn := dg.NewReadOnlyTxn()
	query := `
    {
      q (func: eq(name, "Alice")) {
		name
		dgraph.type
      }
    }`
	resp, err := queryTxn.Query(ctx, query)
	if err != nil {
		log.Fatal("The query should have succeeded")
	}

	fmt.Printf(string(resp.Json))
	// Output: {"q":[]}
}

func ExampleTxn_Mutate_upsert() {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx, toCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer toCancel()

	// Warn: Cleaning up the database
	if err := dg.Alter(ctx, &api.Operation{DropAll: true}); err != nil {
		log.Fatal("The drop all operation should have succeeded")
	}

	op := &api.Operation{}
	op.Schema = `
		name: string .
		email: string @index(exact) .
	`
	if err := dg.Alter(ctx, op); err != nil {
		log.Fatal(err)
	}

	m1 := `
		_:n1 <name> "user" .
		_:n1 <email> "user@dgraphO.io" .
`
	mu := &api.Mutation{
		SetNquads: []byte(m1),
		CommitNow: true,
	}
	if _, err := dg.NewTxn().Mutate(ctx, mu); err != nil {
		log.Fatal(err)
	}

	req := &api.Request{CommitNow: true}
	req.Query = `
		query {
  			me(func: eq(email, "user@dgraphO.io")) {
	    		v as uid
  			}
		}
	`
	m2 := `uid(v) <email> "user@dgraph.io" .`
	mu.SetNquads = []byte(m2)
	req.Mutations = []*api.Mutation{mu}

	// Update email only if matching uid found.
	if _, err := dg.NewTxn().Do(ctx, req); err != nil {
		log.Fatal(err)
	}

	query := `
		{
			me(func: eq(email, "user@dgraph.io")) {
				name
				email
			}
		}
	`
	resp, err := dg.NewTxn().Query(ctx, query)
	if err != nil {
		log.Fatal(err)
	}

	// resp.Json contains the updated value.
	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"user","email":"user@dgraph.io"}]}
}

func ExampleTxn_Mutate_upsertJSON() {
	dg, cancel := getDgraphClient()
	defer cancel()

	// Warn: Cleaning up the database
	ctx := context.Background()
	if err := dg.Alter(ctx, &api.Operation{DropAll: true}); err != nil {
		log.Fatal(err)
	}

	type Person struct {
		Uid     string   `json:"uid,omitempty"`
		Name    string   `json:"name,omitempty"`
		Age     int      `json:"age,omitempty"`
		Email   string   `json:"email,omitempty"`
		Friends []Person `json:"friend,omitempty"`
		DType   string   `json:"dgraph.type,omitempty"`
	}

	op := &api.Operation{Schema: `email: string @index(exact) @upsert .`}
	if err := dg.Alter(context.Background(), op); err != nil {
		log.Fatal(err)
	}

	// Create and query the user using Upsert block
	req := &api.Request{CommitNow: true}
	req.Query = `
		{
			me(func: eq(email, "user@dgraph.io")) {
				...fragmentA
			}
		}

		fragment fragmentA {
			v as uid
		}
	`
	pb, err := json.Marshal(Person{Uid: "uid(v)", Name: "Wrong", Email: "user@dgraph.io"})
	if err != nil {
		log.Fatal(err)
	}
	mu := &api.Mutation{SetJson: pb}
	req.Mutations = []*api.Mutation{mu}
	if _, err := dg.NewTxn().Do(ctx, req); err != nil {
		log.Fatal(err)
	}

	// Fix the name and add age
	pb, err = json.Marshal(Person{Uid: "uid(v)", Name: "user", Age: 35})
	if err != nil {
		log.Fatal(err)
	}
	mu.SetJson = pb
	req.Mutations = []*api.Mutation{mu}
	if _, err := dg.NewTxn().Do(ctx, req); err != nil {
		log.Fatal(err)
	}

	q := `
  		{
			Me(func: has(email)) {
				age
				name
				email
				dgraph.type
			}
		}
	`
	resp, err := dg.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		log.Fatal("The query should have succeeded")
	}

	type Root struct {
		Me []Person `json:"me"`
	}
	var r Root
	if err := json.Unmarshal(resp.Json, &r); err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(resp.Json))

	// Delete the user now
	mu.SetJson = nil
	dgo.DeleteEdges(mu, "uid(v)", "age", "name", "email")
	req.Mutations = []*api.Mutation{mu}
	if _, err := dg.NewTxn().Do(ctx, req); err != nil {
		log.Fatal(err)
	}

	resp, err = dg.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		log.Fatal("The query should have succeeded")
	}
	if err := json.Unmarshal(resp.Json, &r); err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(resp.Json))
	// Output: {"Me":[{"age":35,"name":"user","email":"user@dgraph.io"}]}
	// {"Me":[]}
}
