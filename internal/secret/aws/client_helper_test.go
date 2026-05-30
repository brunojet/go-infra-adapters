package aws

import (
	"errors"
	"strings"
	"testing"

	smTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// ── isNotFound ────────────────────────────────────────────────────────────────

func TestIsNotFound_Nil(t *testing.T) {
	if isNotFound(nil) {
		t.Fatal("expected false for nil error")
	}
}

func TestIsNotFound_TypedResourceNotFoundException(t *testing.T) {
	err := &smTypes.ResourceNotFoundException{}
	if !isNotFound(err) {
		t.Fatal("expected true for ResourceNotFoundException")
	}
}

func TestIsNotFound_StringFallback(t *testing.T) {
	err := errors.New("ResourceNotFoundException: secret not found")
	if !isNotFound(err) {
		t.Fatal("expected true for string-match fallback")
	}
}

func TestIsNotFound_OtherError(t *testing.T) {
	if isNotFound(errors.New("internal server error")) {
		t.Fatal("expected false for unrelated error")
	}
}

// ── marshal / unmarshal ───────────────────────────────────────────────────────

func TestMarshal_Success(t *testing.T) {
	type payload struct {
		Key string `json:"key"`
	}
	b, err := marshal(&payload{Key: "hello"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), "hello") {
		t.Fatalf("unexpected bytes: %s", b)
	}
}

func TestMarshal_Error(t *testing.T) {
	// json.Marshal fails on channel values
	var ch any = make(chan int)
	_, err := marshal(&ch)
	if err == nil || !strings.Contains(err.Error(), "marshal payload") {
		t.Fatalf("expected marshal error, got %v", err)
	}
}

func TestUnmarshal_Success(t *testing.T) {
	type payload struct {
		Key string `json:"key"`
	}
	got, err := unmarshal[payload](`{"key":"val"}`)
	if err != nil || got.Key != "val" {
		t.Fatalf("unmarshal: got=%+v err=%v", got, err)
	}
}

func TestUnmarshal_Error(t *testing.T) {
	_, err := unmarshal[struct{ Key string }]("not-json")
	if err == nil || !strings.Contains(err.Error(), "unmarshal payload") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
}
