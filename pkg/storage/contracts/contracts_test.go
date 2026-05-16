package contracts

import (
	"io"
	"testing"
)

type dummyCloser struct{ closed bool }

func (d *dummyCloser) Read(p []byte) (int, error) { return 0, io.EOF }
func (d *dummyCloser) Close() error               { d.closed = true; return nil }

func TestBucketObject_Close_NilAndBody(t *testing.T) {
	var bo *BucketObject
	if err := bo.Close(); err != nil {
		t.Fatalf("expected nil error for nil receiver, got %v", err)
	}

	bo = &BucketObject{}
	if err := bo.Close(); err != nil {
		t.Fatalf("expected nil error for nil body, got %v", err)
	}

	dc := &dummyCloser{}
	bo = &BucketObject{Body: dc}
	if err := bo.Close(); err != nil {
		t.Fatalf("expected nil error closing body, got %v", err)
	}
	if !dc.closed {
		t.Fatalf("expected underlying closer to be closed")
	}
}
