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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
)

var (
	username     = "alice"
	userpassword = "alicepassword"
	readpred     = "predicate_to_read"
	writepred    = "predicate_to_write"
	modifypred   = "predicate_to_modify"
	unusedgroup  = "unused"
	devgroup     = "dev"

	dgraphAddress = "127.0.0.1:9180"
	adminUrl      = "http://127.0.0.1:8180/admin"
)

func initializeDBACLs(t *testing.T, dg *dgo.Dgraph) {
	// Clean up DB.
	op := &api.Operation{}
	op.DropAll = true
	err := dg.Alter(context.Background(), op)
	require.NoError(t, err, "unable to drop data for ACL tests")

	// Create schema for read predicate.
	op = &api.Operation{}
	op.Schema = fmt.Sprintf("%s: string @index(exact) .", readpred)
	err = dg.Alter(context.Background(), op)
	require.NoError(t, err, "unable to insert schema for read predicate")

	// Insert some record to read for read predicate.
	data := []byte(fmt.Sprintf(`_:sub <%s> "val1" .`, readpred))
	_, err = dg.NewTxn().Mutate(context.Background(), &api.Mutation{SetNquads: data})
	require.NoError(t, err, "unable to insert data for read predicate")
}

func resetUser(t *testing.T, token *HttpToken) {
	resetUser := `mutation addUser($name: String!, $pass: String!) {
					  addUser(input: [{name: $name, password: $pass}]) {
						user {
						  name
						}
					  }
					}`

	params := &GraphQLParams{
		Query: resetUser,
		Variables: map[string]interface{}{
			"name": username,
			"pass": userpassword,
		},
	}
	resp := MakeGQLRequest(t, adminUrl, params, token)
	require.True(t, len(resp.Errors) == 0, resp.Errors)
}

func createGroupACLs(t *testing.T, groupname string, token *HttpToken) {
	// create group
	createGroup := `mutation addGroup($name: String!){
			addGroup(input: [{name: $name}]) {
				group {
				  name
				  users {
					name
				  }
				}
			  }
			}`
	params := &GraphQLParams{
		Query: createGroup,
		Variables: map[string]interface{}{
			"name": groupname,
		},
	}
	resp := MakeGQLRequest(t, adminUrl, params, token)
	require.Truef(t, len(resp.Errors) == 0, "unable to create group: %+v", resp.Errors.Error())

	// Set permissions.
	updatePerms := `mutation updateGroup($gname: String!, $pred: String!, $perm: Int!) {
		  updateGroup(input: {filter: {name: {eq: $gname}}, set: {rules: [{predicate: $pred, permission: $perm}]}}) {
			group {
			  name
			  rules {
				permission
				predicate
			  }
			}
		  }
		}`

	setPermission := func(pred string, permission int) {
		params = &GraphQLParams{
			Query: updatePerms,
			Variables: map[string]interface{}{
				"gname": groupname,
				"pred":  pred,
				"perm":  permission,
			},
		}
		resp = MakeGQLRequest(t, adminUrl, params, token)
		require.Truef(t, len(resp.Errors) == 0, "unable to set permissions: %+v", resp.Errors)
	}

	// assign read access to read predicate
	setPermission(readpred, 4)
	// assign write access to write predicate
	setPermission(writepred, 2)
	// assign modify access to modify predicate
	setPermission(modifypred, 1)
}

func addUserToGroup(t *testing.T, username, group, op string, token *HttpToken) {
	addToGroup := `mutation updateUser($name: String, $group: String!) {
		updateUser(input: {filter: {name: {eq: $name}}, set: {groups: [{name: $group}]}}) {
			user {
			  name
			  groups {
				name
			}
			}
		  }
		}`
	removeFromGroup := `mutation updateUser($name: String, $group: String!) {
		updateUser(input: {filter: {name: {eq: $name}}, remove: {groups: [{name: $group}]}}) {
			user {
			  name
			  groups {
				name
			}
			}
		  }
		}`

	var query string
	switch op {
	case "add":
		query = addToGroup
	case "del":
		query = removeFromGroup
	default:
		require.Fail(t, "invalid operation for updating user")
	}

	params := &GraphQLParams{
		Query: query,
		Variables: map[string]interface{}{
			"name":  username,
			"group": group,
		},
	}
	resp := MakeGQLRequest(t, adminUrl, params, token)
	require.Truef(t, len(resp.Errors) == 0, "unable to update user: %+v", resp.Errors)
}

func query(t *testing.T, dg *dgo.Dgraph, shouldFail bool) {
	q := `
	{
		q(func: eq(predicate_to_read, "val1")) {
			predicate_to_read
		}
	}
	`

	// Dgraph does not throw Permission Denied error in query any more. Dgraph
	// just does not return the predicates that a user doesn't have access to.
	resp, err := dg.NewReadOnlyTxn().Query(context.Background(), q)
	require.NoError(t, err)
	if shouldFail {
		require.Equal(t, string(resp.Json), "{}")
	}
}

func mutation(t *testing.T, dg *dgo.Dgraph, shouldFail bool) {
	mu := &api.Mutation{
		SetNquads: []byte(fmt.Sprintf(`_:uid <%s> "val2" .`, writepred)),
	}

	_, err := dg.NewTxn().Mutate(context.Background(), mu)
	if (err != nil && !shouldFail) || (err == nil && shouldFail) {
		t.Logf("result did not match for mutation")
		t.FailNow()
	}
}

func changeSchema(t *testing.T, dg *dgo.Dgraph, shouldFail bool) {
	op := &api.Operation{
		Schema: fmt.Sprintf("%s: string @index(exact) .", modifypred),
	}

	err := dg.Alter(context.Background(), op)
	if (err != nil && !shouldFail) || (err == nil && shouldFail) {
		t.Logf("result did not match for schema change")
		t.FailNow()
	}
}

func TestACLs(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	initializeDBACLs(t, dg)

	token, err := HttpLogin(&LoginParams{
		Endpoint:  adminUrl,
		UserID:    "groot",
		Passwd:    "password",
		Namespace: 0, // Galaxy namespace
	})
	resetUser(t, token)
	time.Sleep(5 * time.Second)

	// All operations without ACLs should fail.
	err = dg.Login(context.Background(), username, userpassword)
	require.NoError(t, err, "unable to login for user: %s", username)
	query(t, dg, true)
	mutation(t, dg, true)
	changeSchema(t, dg, true)

	// Create unused group, everything should still fail.
	createGroupACLs(t, unusedgroup, token)
	time.Sleep(6 * time.Second)
	query(t, dg, true)
	mutation(t, dg, true)
	changeSchema(t, dg, true)

	// Create dev group and link user to it. Everything should pass now.
	createGroupACLs(t, devgroup, token)
	addUserToGroup(t, username, devgroup, "add", token)
	time.Sleep(6 * time.Second)
	query(t, dg, false)
	mutation(t, dg, false)
	changeSchema(t, dg, false)

	// Remove user from dev group, everything should fail now.
	addUserToGroup(t, username, devgroup, "del", token)
	time.Sleep(6 * time.Second)
	query(t, dg, true)
	mutation(t, dg, true)
	changeSchema(t, dg, true)
}
