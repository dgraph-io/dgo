/*
 * Copyright 2018 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package test

import (
	"fmt"
	"github.com/dgraph-io/dgo"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/dgraph-io/dgo/protos/api"
	"github.com/dgraph-io/dgraph/x"
	"google.golang.org/grpc"
)

type DgraphCluster struct {
	TokenizerPluginsArg string

	DgraphPort string
	ZeroPort   string

	DgraphPortOffset int
	ZeroPortOffset   int

	Dir    string
	Zero   *exec.Cmd
	Dgraph *exec.Cmd

	Client *dgo.Dgraph
}

func init() {
	cmd := exec.Command("go", "install", "github.com/dgraph-io/dgraph/dgraph")
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("Could not run %q: %s", cmd.Args, string(out))
	}
}

func FreePort(port int) int {
	// Linux reuses ports in FIFO order. So a port that we listen on and then
	// release will be free for a long time.
	for {
		// p + 5080 and p + 9080 must lie within [20000, 60000]
		offset := 15000 + rand.Intn(30000)
		p := port + offset
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			listener.Close()
			return offset
		}
	}
}


func NewDgraphCluster(dir string) *DgraphCluster {
	do := FreePort(x.PortGrpc)
	zo := FreePort(x.PortZeroGrpc)
	return &DgraphCluster{
		DgraphPort:       strconv.Itoa(do + x.PortGrpc),
		ZeroPort:         strconv.Itoa(zo + x.PortZeroGrpc),
		DgraphPortOffset: do,
		ZeroPortOffset:   zo,
		Dir:              dir,
	}
}

func (d *DgraphCluster) StartZeroOnly() error {
	d.Zero = exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"zero",
		"-w=wz",
		"-o", strconv.Itoa(d.ZeroPortOffset),
		"--replicas", "3",
	)
	d.Zero.Dir = d.Dir
	d.Zero.Stdout = nil
	d.Zero.Stderr = nil

	if err := d.Zero.Start(); err != nil {
		return err
	}

	// Wait for dgraphzero to start listening and become the leader.
	time.Sleep(time.Second * 4)
	return nil
}

func (d *DgraphCluster) Start() error {
	if err := d.StartZeroOnly(); err != nil {
		return err
	}

	d.Dgraph = exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"alpha",
		"--lru_mb=4096",
		"--zero", ":"+d.ZeroPort,
		"--port_offset", strconv.Itoa(d.DgraphPortOffset),
		"--custom_tokenizers", d.TokenizerPluginsArg,
	)
	d.Dgraph.Dir = d.Dir
	d.Dgraph.Stdout = nil
	d.Dgraph.Stderr = nil
	if err := d.Dgraph.Start(); err != nil {
		return err
	}

	dgConn, err := grpc.Dial(":"+d.DgraphPort, grpc.WithInsecure())
	if err != nil {
		return err
	}

	// Wait for Dgraph to start accepting requests. TODO: Could do this
	// programmatically by hitting the query port. This would be quicker than
	// just waiting 4 seconds (which seems to be the smallest amount of time to
	// reliably wait).
	time.Sleep(time.Second * 4)

	d.Client = dgo.NewDgraphClient(api.NewDgraphClient(dgConn))

	return nil
}

type Node struct {
	Process *exec.Cmd
	Offset  string
}

func (d *DgraphCluster) AddNode(dir string) (Node, error) {
	o := strconv.Itoa(FreePort(x.PortInternal))
	dgraph := exec.Command(os.ExpandEnv("$GOPATH/bin/dgraph"),
		"alpha",
		"--lru_mb=4096",
		"--zero", ":"+d.ZeroPort,
		"--port_offset", o,
	)
	dgraph.Dir = dir
	dgraph.Stdout = os.Stdout
	dgraph.Stderr = os.Stderr
	x.Check(os.MkdirAll(dir, os.ModePerm))
	err := dgraph.Start()

	return Node{
		Process: dgraph,
		Offset:  o,
	}, err
}

func (d *DgraphCluster) Close() {
	// Ignore errors
	if d.Zero != nil && d.Zero.Process != nil {
		d.Zero.Process.Kill()
	}
	if d.Dgraph != nil && d.Dgraph.Process != nil {
		d.Dgraph.Process.Kill()
	}
}

func MakeDirEmpty(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}