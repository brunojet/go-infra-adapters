package main

import (
	"context"
	"fmt"

	_ "github.com/brunojet/go-infra-adapters/internal/secrets/aws"
	"github.com/brunojet/go-infra-adapters/pkg/providers"
)

var p providers.Provider

func init() {
	var ok bool
	p, ok = providers.GetProvider("aws")
	if !ok {
		panic("aws provider not registered")
	}
}

func main() {
	secretAPI := p.Secret()
	if secretAPI == nil {
		fmt.Println("provider 'aws' does not expose Secret feature")
		return
	}

	adapter, err := secretAPI.NewSecret("example-secret")
	if err != nil {
		fmt.Println("failed to get secret adapter:", err)
		return
	}

	val, err := adapter.GetCurrent(context.Background())
	if err != nil {
		fmt.Println("failed to get secret:", err)
		return
	}

	fmt.Printf("secret data length: %d metadata: %#v\n", len(val.Data), val.Metadata)
}
