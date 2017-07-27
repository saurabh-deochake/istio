// Copyright 2017 Istio Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Adapted from istio/proxy/test/backend/echo with error handling and
// concurrency fixes and making it as low overhead as possible
// (no std output by default)

package main

import (
	"flag"
	"fmt"
	"net"

	"istio.io/istio/devel/fortio"

	context "golang.org/x/net/context"

	"google.golang.org/grpc"
)

var (
	countFlag   = flag.Int("n", 1, "how many ping the client will send")
	portFlag    = flag.Int("port", 8079, "default grpc port")
	hostFlag    = flag.String("host", "", "client mode: server to connect to")
	payloadFlag = flag.String("payload", "this is the default payload", "Payload string to send along")
)

type pingServer struct {
}

func (s *pingServer) Ping(c context.Context, in *PingMessage) (*PingMessage, error) {
	fortio.Infof("Ping called %+v (ctx %+v)", *in, c)
	out := *in
	out.Ttl++
	return &out, nil
}

func startServer(port int) {
	socket, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fortio.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	RegisterPingServerServer(grpcServer, &pingServer{})
	fmt.Printf("Fortio grpc ping server listening on port %v\n", port)
	if err := grpcServer.Serve(socket); err != nil {
		fortio.Fatalf("failed to start grpc server: %v", err)
	}
}

func clientCall(serverAddr string, n int, payload string) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		fortio.Fatalf("failed to conect to %s: %v", serverAddr, err)
	}
	msg := &PingMessage{}
	msg.Payload = payload
	cli := NewPingServerClient(conn)
	for i := 1; i <= n; i++ {
		msg.Id = int64(i)
		res, err := cli.Ping(context.Background(), msg)
		if err != nil {
			fortio.Fatalf("grpc error from Ping %v", err)
		}
		fortio.Infof("Ping returned %+v", *res)
		msg = res
	}
}

func main() {
	flag.Parse()
	if *hostFlag != "" {
		// TODO doesn't work for ipv6 addrs etc
		clientCall(fmt.Sprintf("%s:%d", *hostFlag, *portFlag), *countFlag, *payloadFlag)
	} else {
		startServer(*portFlag)
	}
}