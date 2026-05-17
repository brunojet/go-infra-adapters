// Package main is a small example binary that demonstrates retrieving a
// secret via the registered provider adapter. It is not used in tests but
// kept as an executable example.
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	_ "github.com/brunojet/go-infra-adapters/internal/secret/aws"
	providers "github.com/brunojet/go-infra-adapters/pkg/provider"
	storagecontracts "github.com/brunojet/go-infra-adapters/pkg/storage/contracts"
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

	// --- Storage example demonstrating ContentType + Head/Get/Put ---
	storageAPI := p.Storage()
	if storageAPI == nil {
		fmt.Println("provider 'aws' does not expose Storage feature")
		return
	}

	bkt, err := storageAPI.NewBucket("example-bucket")
	if err != nil {
		fmt.Println("failed to get bucket adapter:", err)
		return
	}

	// Prepare content and a BucketObject including ContentType.
	data := []byte("hello storage example")
	put := &storagecontracts.BucketObject{Info: storagecontracts.ObjectInfo{Key: "example.txt", Size: int64(len(data)), ContentType: "text/plain"}, Body: io.NopCloser(bytes.NewReader(data))}
	if err := bkt.PutObject(context.Background(), put); err != nil {
		fmt.Println("PutObject failed:", err)
		return
	}

	// Inspect metadata via HeadObject (fills ObjectInfo)
	var info storagecontracts.ObjectInfo
	if err := bkt.HeadObject(context.Background(), "example.txt", &info); err != nil {
		fmt.Println("HeadObject failed:", err)
		return
	}
	fmt.Printf("Head: key=%s size=%d content-type=%s metadata=%#v\n", info.Key, info.Size, info.ContentType, info.Metadata)

	// Download using GetObject (fills BucketObject)
	got := &storagecontracts.BucketObject{}
	if err := bkt.GetObject(context.Background(), "example.txt", got); err != nil {
		fmt.Println("GetObject failed:", err)
		return
	}
	defer func() { _ = got.Close() }()
	body, _ := io.ReadAll(got.Body)
	fmt.Printf("Get: key=%s content-type=%s data=%s\n", got.Info.Key, got.Info.ContentType, string(body))
}
