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

package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dgraph-io/dgraph/ee/acl"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/stretchr/testify/require"
)

func TestAcl(t *testing.T) {
	// setup a cluster
	//cluster := setupCluster()
	//defer cluster.Close()
	dg, close := GetDgraphClient()
	defer close()

	createAccountAndData(t, dg)
	queryPredicateWithUserAccount(t, dg, true)
	createGroupAndAcls(t)
	// wait for 35 seconds to ensure the new acl have reached all acl caches
	// on all alpha servers
	log.Println("Sleeping for 35 seconds for acl to catch up")
	time.Sleep(35 * time.Second)
	queryPredicateWithUserAccount(t, dg, false)
}

var user = "alice"
var password = "password123"
var predicate = "city_name"
var group = "dev"
var rootDir = filepath.Join(os.TempDir(), "acl_test")

func queryPredicateWithUserAccount(t *testing.T, dg *dgo.Dgraph, shouldFail bool) {
	// try to query the user whose name is alice
	ctx := context.Background()
	if err := dg.Login(ctx, user, password); err != nil {
		t.Fatalf("unable to login using the account %v", user)
	}

	ctxWithUserJwt := dg.GetContext(ctx)
	txn := dg.NewTxn()
	query := fmt.Sprintf(`
	{
		q(func: eq(%s, "SF")) {
			name
		}
	}`, predicate)
	txn = dg.NewTxn()
	_, err := txn.Query(ctxWithUserJwt, query)

	if shouldFail {
		require.Error(t, err, "the query should have failed")
	} else {
		require.NoError(t, err, "the query should have succeeded")
	}
}

func createAccountAndData(t *testing.T, dg *dgo.Dgraph) {
	// use the admin account to clean the database
	ctx := context.Background()
	if err := dg.Login(ctx, "admin", "password"); err != nil {
		t.Fatalf("unable to login using the admin account")
	}
	ctxWithAdminJwt := dg.GetContext(ctx)
	op := api.Operation{
		DropAll: true,
	}
	if err := dg.Alter(ctxWithAdminJwt, &op); err != nil {
		t.Fatalf("Unable to cleanup db:%v", err)
	}

	// use commands to create users and groups
	createUserCmd := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "useradd",
		"-d", "localhost:9180",
		"-u", user, "-p", password, "--adminPassword", "password")
	if err := createUserCmd.Run(); err != nil {
		t.Fatalf("Unable to create user:%v", err)
	}

	// create some data, e.g. user with name alice
	require.NoError(t, dg.Alter(ctxWithAdminJwt, &api.Operation{
		Schema: `city_name: string @index(exact) .`,
	}))

	txn := dg.NewTxn()
	_, err := txn.Mutate(ctxWithAdminJwt, &api.Mutation{
		SetNquads: []byte(fmt.Sprintf("_:a <%s> \"SF\" .", predicate)),
	})
	require.NoError(t, err)
	require.NoError(t, txn.Commit(ctx))
}

func createGroupAndAcls(t *testing.T) {
	// use commands to create users and groups
	createGroupCmd := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "groupadd",
		"-d", "localhost:9180",
		"-g", group, "--adminPassword", "password")
	if err := createGroupCmd.Run(); err != nil {
		t.Fatalf("Unable to create group:%v", err)
	}

	addPermCmd1 := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "chmod",
		"-d", "localhost:9180",
		"-g", group, "-p", predicate, "-P", strconv.Itoa(int(acl.Read)), "--adminPassword",
		"password")
	if err := addPermCmd1.Run(); err != nil {
		t.Fatalf("Unable to add permission to group %s:%v", group, err)
	}

	addPermCmd2 := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "chmod",
		"-d", "localhost:9180",
		"-g", group, "-p", "name", "-P", strconv.Itoa(int(acl.Read)), "--adminPassword",
		"password")
	if err := addPermCmd2.Run(); err != nil {
		t.Fatalf("Unable to add permission to group %s:%v", group, err)
	}


	addUserToGroupCmd := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"acl", "usermod",
		"-d", "localhost:9180",
		"-u", user, "-g", group, "--adminPassword", "password")
	if err := addUserToGroupCmd.Run(); err != nil {
		t.Fatalf("Unable to add user %s to group %s:%v", user, group, err)
	}
}

/*
func setupCluster() *DgraphCluster {
	if err := MakeDirEmpty(rootDir); err != nil {
		log.Fatalf("Unable to create dir %v", rootDir)
	}

	cluster := NewDgraphCluster(rootDir)
	if err := cluster.Start(); err != nil {
		log.Fatalf("Unable to start cluster: %v", err)
	}
	return cluster
}
*/
