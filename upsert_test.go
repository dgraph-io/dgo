/*
 * Copyright (C) 2019 Dgraph Labs, Inc. and Contributors
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
	"sort"
	"testing"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/stretchr/testify/require"
)

func TestCondUpsertCorrectingName(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `email: string @index(exact) .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	// Erroneously, mutation with wrong name "wrong"
	q1 := `
{
  me(func: eq(email, "email@company.io")) {
	v as uid
  }
}`
	m1 := `
[
  {
    "uid": "uid(v)",
    "name": "Wrong"
  },
  {
    "uid": "uid(v)",
    "email": "email@company.io"
  }
]`
	req := &api.Request{
		CommitNow: true,
		Query:     q1,
		Mutations: []*api.Mutation{
			&api.Mutation{
				Cond:    ` @if(eq(len(v), 0)) `,
				SetJson: []byte(m1),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	// query should return the wrong name
	q2 := `
{
  q(func: has(email)) {
    uid
    name
    email
  }
}`
	req = &api.Request{Query: q2}
	resp, err := dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)
	require.Contains(t, string(resp.Json), "Wrong")

	// Fixing the name in the database, mutation with correct name
	q3 := q1
	m3 := `
[
  {
    "uid": "uid(v)",
    "name": "Ashish"
  }
]`
	req = &api.Request{
		CommitNow: true,
		Query:     q3,
		Mutations: []*api.Mutation{
			&api.Mutation{
				Cond:    ` @if(eq(len(v), 1)) `,
				SetJson: []byte(m3),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	// query should return correct name
	req = &api.Request{Query: q2}
	resp, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)
	require.Contains(t, string(resp.Json), "Ashish")
}

type Employee struct {
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	WorksFor  string     `json:"works_for"`
	WorksWith []Employee `json:"works_with"`
}

func populateCompanyData(t *testing.T, dg *dgo.Dgraph) {
	op := &api.Operation{}
	op.Schema = `
email: string @index(exact) .
works_for: string @index(exact) .
works_with: [uid] .`
	err := dg.Alter(context.Background(), op)
	require.NoError(t, err)

	m1 := `
_:user1 <name> "user1" .
_:user1 <email> "user1@company1.io" .
_:user1 <works_for> "company1" .

_:user2 <name> "user2" .
_:user2 <email> "user2@company1.io" .
_:user2 <works_for> "company1" .

_:user3 <name> "user3" .
_:user3 <email> "user3@company2.io" .
_:user3 <works_for> "company2" .

_:user4 <name> "user4" .
_:user4 <email> "user4@company2.io" .
_:user4 <works_for> "company2" .`
	req := &api.Request{
		CommitNow: true,
		Mutations: []*api.Mutation{
			&api.Mutation{
				SetNquads: []byte(m1),
			},
		},
	}
	_, err = dg.NewTxn().Do(context.Background(), req)
	require.NoError(t, err)
}

func TestUpsertMultiValueEdge(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	populateCompanyData(t, dg)

	// All employees of company1 now work with all employees of company2
	q1 := `
{
  c1 as var(func: eq(works_for, "company1"))
  c2 as var(func: eq(works_for, "company2"))
}`
	m1 := `
uid(c1) <works_with> uid(c2) .
uid(c2) <works_with> uid(c1) .`
	req := &api.Request{
		CommitNow: true,
		Query:     q1,
		Mutations: []*api.Mutation{
			&api.Mutation{
				Cond:      `@if(eq(len(c1), 2) AND eq(len(c2), 2))`,
				SetNquads: []byte(m1),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	q2 := `
{
  q(func: eq(works_for, "%s")) {
    name
    works_with {
      name
    }
  }
}`
	req = &api.Request{Query: fmt.Sprintf(q2, "company1")}
	resp, err := dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)
	var res1 struct {
		Employees []Employee `json:"q"`
	}
	err = json.Unmarshal(resp.Json, &res1)
	require.NoError(t, err)
	cls := []string{res1.Employees[0].WorksWith[0].Name, res1.Employees[0].WorksWith[1].Name}
	sort.Strings(cls)
	require.Equal(t, []string{"user3", "user4"}, cls)
	cls = []string{res1.Employees[1].WorksWith[0].Name, res1.Employees[1].WorksWith[1].Name}
	sort.Strings(cls)
	require.Equal(t, []string{"user3", "user4"}, cls)

	req = &api.Request{Query: fmt.Sprintf(q2, "company2")}
	resp, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)
	var res2 struct {
		Employees []Employee `json:"q"`
	}
	err = json.Unmarshal(resp.Json, &res2)
	require.NoError(t, err)
	cls = []string{res2.Employees[0].WorksWith[0].Name, res2.Employees[0].WorksWith[1].Name}
	sort.Strings(cls)
	require.Equal(t, []string{"user1", "user2"}, cls)
	cls = []string{res2.Employees[1].WorksWith[0].Name, res2.Employees[1].WorksWith[1].Name}
	sort.Strings(cls)
	require.Equal(t, []string{"user1", "user2"}, cls)
}

func TestUpsertEdgeWithBlankNode(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	populateCompanyData(t, dg)

	// Add a new employee who works with every employee in company2
	q1 := `
{
  c1 as var(func: eq(works_for, "company1"))
  c2 as var(func: eq(works_for, "company2"))
}`
	m1 := `
_:user5 <name> "user5" .
_:user5 <email> "user5@company1.io" .
_:user5 <works_for> "company1" .
_:user5 <works_with> uid(c2) .`
	req := &api.Request{
		CommitNow: true,
		Query:     q1,
		Mutations: []*api.Mutation{
			&api.Mutation{
				Cond:      `@if(lt(len(c1), 3))`,
				SetNquads: []byte(m1),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	q2 := `
{
  q(func: eq(email, "user5@company1.io")) {
    name
    email
    works_for
    works_with {
      name
    }
  }
}`
	req = &api.Request{Query: q2}
	resp, err := dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	var res struct {
		Employees []Employee `json:"q"`
	}
	err = json.Unmarshal(resp.Json, &res)
	require.NoError(t, err)

	require.Equal(t, 1, len(res.Employees))
	v := res.Employees[0]
	require.Equal(t, "user5", v.Name)
	require.Equal(t, "user5@company1.io", v.Email)
	require.Equal(t, "company1", v.WorksFor)
	require.Equal(t, 2, len(v.WorksWith))
	cls := []string{v.WorksWith[0].Name, v.WorksWith[1].Name}
	sort.Strings(cls)
	require.Equal(t, []string{"user3", "user4"}, cls)
}

func TestUpsertDeleteOnlyYourPost(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `
name: string @index(exact) .
content: string @index(exact) .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	m1 := `
_:user1 <name> "user1" .
_:user2 <name> "user2" .
_:user3 <name> "user3" .
_:user4 <name> "user4" .

_:post1 <content> "post1" .
_:post1 <author> _:user1 .

_:post2 <content> "post2" .
_:post2 <author> _:user1 .

_:post3 <content> "post3" .
_:post3 <author> _:user2 .

_:post4 <content> "post4" .
_:post4 <author> _:user3 .

_:post5 <content> "post5" .
_:post5 <author> _:user3 .

_:post6 <content> "post6" .
_:post6 <author> _:user3 .
`
	req := &api.Request{
		CommitNow: true,
		Mutations: []*api.Mutation{
			&api.Mutation{
				SetNquads: []byte(m1),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	// user2 trying to delete the post4
	q2 := `
{
  var(func: eq(content, "post4")) {
    p4 as uid
    author {
      n3 as name
    }
  }

  u2 as var(func: eq(val(n3), "user2"))
}`
	m2 := `
uid(p4) <content> * .
uid(p4) <author> * .`
	req = &api.Request{
		CommitNow: true,
		Query:     q2,
		Mutations: []*api.Mutation{
			&api.Mutation{
				Cond:      `@if(eq(len(u2), 1))`,
				DelNquads: []byte(m2),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	q3 := `
{
  post(func: eq(content, "post4")) {
    content
  }
}`
	req = &api.Request{Query: q3}
	resp, err := dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)
	require.Contains(t, string(resp.Json), "post4")

	// user3 deleting the post4
	q4 := `
{
  var(func: eq(content, "post4")) {
    p4 as uid
    author {
      n3 as name
    }
  }

  u4 as var(func: eq(val(n3), "user3"))
}`
	m4 := `
uid(p4) <content> * .
uid(p4) <author> * .`
	req = &api.Request{
		CommitNow: true,
		Query:     q4,
		Mutations: []*api.Mutation{
			&api.Mutation{
				Cond:      `@if(eq(len(u4), 1))`,
				DelNquads: []byte(m4),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	req = &api.Request{Query: q3}
	resp, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)
	require.NotContains(t, string(resp.Json), "post4")
}

func TestUpsertBulkUpdateBranch(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `
name: string @index(exact) .
branch: string .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	m1 := `
_:user1 <name> "user1" .
_:user1 <branch> "Fuller Street, San Francisco" .

_:user2 <name> "user2" .
_:user2 <branch> "Fuller Street, San Francisco" .

_:user3 <name> "user3" .
_:user3 <branch> "Fuller Street, San Francisco" .
`
	req := &api.Request{
		CommitNow: true,
		Mutations: []*api.Mutation{
			&api.Mutation{
				SetNquads: []byte(m1),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	// Bulk Update: update everyone's branch
	req = &api.Request{
		CommitNow: true,
		Query:     `{ u as var(func: has(branch)) }`,
		Mutations: []*api.Mutation{
			&api.Mutation{
				SetNquads: []byte(`uid(u) <branch> "Fuller Street, SF" .`),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	q1 := `
{
  q(func: has(branch)) {
    name
    branch
  }
}`
	req = &api.Request{Query: q1}
	resp, err := dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	var res1, res2 struct {
		Q []struct {
			Branch string
		}
	}
	err = json.Unmarshal(resp.Json, &res1)
	require.NoError(t, err)
	for _, v := range res1.Q {
		require.Equal(t, "Fuller Street, SF", v.Branch)
	}

	// Bulk Delete: delete everyone's branch
	req = &api.Request{
		CommitNow: true,
		Query:     `{ u as var(func: has(branch)) }`,
		Mutations: []*api.Mutation{
			&api.Mutation{
				DelNquads: []byte(`uid(u) <branch> * .`),
			},
		},
	}
	_, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	req = &api.Request{Query: q1}
	resp, err = dg.NewTxn().Do(ctx, req)
	require.NoError(t, err)

	err = json.Unmarshal(resp.Json, &res2)
	require.NoError(t, err)
	for _, v := range res2.Q {
		require.Nil(t, v.Branch)
	}
}

func TestBulkDelete(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `email: string @index(exact) .
	name: string @index(exact) .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	// Insert 2 users.
	mu1 := &api.Mutation{
		SetNquads: []byte(`
		_:alice <name> "alice" .
		_:alice <email> "alice@company1.io" .
		_:bob <name> "bob" .
		_:bob <email> "bob@company1.io" .`),
	}
	req1 := &api.Request{
		Mutations: []*api.Mutation{mu1},
		CommitNow: true,
	}
	_, err = dg.NewTxn().Do(context.Background(), req1)
	require.NoError(t, err, "unable to load data")

	// Delete all data for user alice.
	q2 := `{
		v as var(func: eq(name, "alice"))
	}`
	mu2 := &api.Mutation{
		DelNquads: []byte(`
		uid(v) <name> * .
		uid(v) <email> * .`),
	}
	req2 := &api.Request{
		CommitNow: true,
		Query:     q2,
		Mutations: []*api.Mutation{mu2},
	}
	_, err = dg.NewTxn().Do(context.Background(), req2)
	require.NoError(t, err, "unable to perform delete")

	// Get record with email.
	q3 := `{
	q(func: has(email)) {
		email
	}
	}`
	req3 := &api.Request{Query: q3}
	res, err := dg.NewTxn().Do(context.Background(), req3)
	require.NoError(t, err, "unable to query after bulk delete")

	var res1 struct {
		Q []struct {
			Email string `json:"email"`
		} `json:"q"`
	}
	require.NoError(t, json.Unmarshal(res.Json, &res1))
	require.Equal(t, res1.Q[0].Email, "bob@company1.io")
}
