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
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
)

var (
	grootpassword = "password"
	username      = "alice"
	userpassword  = "alicepassword"
	readpred      = "predicate_to_read"
	writepred     = "predicate_to_write"
	modifypred    = "predicate_to_modify"
	unusedgroup   = "unused"
	devgroup      = "dev"

	dgraphAddress = "127.0.0.1:9180"
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

func resetUser(t *testing.T) {
	createuser := exec.Command("dgraph", "acl", "add", "-a", dgraphAddress, "-u", username, "-p",
		userpassword, "-x", grootpassword)

	_, err := createuser.CombinedOutput()
	require.NoError(t, err, "unable to create user")
}

func createGroupACLs(t *testing.T, groupname string) {
	// create group
	createGroup := exec.Command("dgraph", "acl", "add", "-a", dgraphAddress,
		"-g", groupname, "-x", grootpassword)
	_, err := createGroup.CombinedOutput()
	require.NoError(t, err, "unable to create group: %s", groupname)

	// assign read access to read predicate
	readAccess := exec.Command("dgraph", "acl", "mod", "-a", dgraphAddress, "-g", groupname, "-p",
		readpred, "-m", "4", "-x", grootpassword)
	_, err = readAccess.CombinedOutput()
	require.NoError(t, err, "unable to grant read access to group: %s", groupname)

	// assign write access to write predicate
	writeAccess := exec.Command("dgraph", "acl", "mod", "-a", dgraphAddress, "-g", groupname, "-p",
		writepred, "-m", "2", "-x", grootpassword)
	_, err = writeAccess.CombinedOutput()
	require.NoError(t, err, "unable to grant write access to group: %s", groupname)

	// assign modify access to modify predicate
	modifyAccess := exec.Command("dgraph", "acl", "mod", "-a", dgraphAddress, "-g", groupname, "-p",
		modifypred, "-m", "1", "-x", grootpassword)
	_, err = modifyAccess.CombinedOutput()
	require.NoError(t, err, "unable to grant modify access to group: %s", groupname)
}

func addUserToGroup(t *testing.T, username, groupname string) {
	linkCmd := exec.Command("dgraph", "acl", "mod", "-a", dgraphAddress, "-u", username,
		"--group_list", groupname, "-x", grootpassword)
	_, err := linkCmd.CombinedOutput()
	require.NoError(t, err, "unable to link user: %s and group: %s", username, groupname)
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

	resetUser(t)
	time.Sleep(5 * time.Second)

	// All operations without ACLs should fail.
	err := dg.Login(context.Background(), username, userpassword)
	require.NoError(t, err, "unable to login for user: %s", username)
	query(t, dg, true)
	mutation(t, dg, true)
	changeSchema(t, dg, true)

	// Create unused group, everything should still fail.
	createGroupACLs(t, unusedgroup)
	time.Sleep(6 * time.Second)
	query(t, dg, true)
	mutation(t, dg, true)
	changeSchema(t, dg, true)

	// Create dev group and link user to it. Everything should pass now.
	createGroupACLs(t, devgroup)
	addUserToGroup(t, username, devgroup)
	time.Sleep(6 * time.Second)
	query(t, dg, false)
	mutation(t, dg, false)
	changeSchema(t, dg, false)

	// Remove user from dev group, everything should fail now.
	addUserToGroup(t, username, "")
	time.Sleep(6 * time.Second)
	query(t, dg, true)
	mutation(t, dg, true)
	changeSchema(t, dg, true)
}
