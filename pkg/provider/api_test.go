package provider

import "testing"

func TestProviderApi_WrapperCalls(t *testing.T) {
	// call for non-existent provider
	if p, ok := GetProvider("__does_not_exist__"); ok || p != nil {
		t.Fatalf("expected no provider")
	}
	_ = SupportedProviders()
}

func TestExists_pkg_provider_api(t *testing.T) {}
