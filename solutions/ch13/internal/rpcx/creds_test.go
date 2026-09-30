package rpcx_test

import (
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// insecureCreds is plaintext gRPC, fine for tests on loopback. Between
// machines you would use TLS (or mutual TLS between services).
func insecureCreds() credentials.TransportCredentials { return insecure.NewCredentials() }
