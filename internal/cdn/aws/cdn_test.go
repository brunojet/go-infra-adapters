package aws

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	goaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/golang/mock/gomock"

	mockcf "github.com/brunojet/go-infra-adapters/v3/internal/cdn/aws/mock"
	"github.com/brunojet/go-infra-adapters/v3/pkg/cdn/contracts"
)

func newMock(t *testing.T) (*gomock.Controller, *mockcf.MockCloudFrontClient, *cdnAdapter) {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := mockcf.NewMockCloudFrontClient(ctrl)
	dist := NewCdn(WithClient(client))
	return ctrl, client, dist
}

// --- options ---

func TestDefaults(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	_ = client
	if dist.maxKeys != 3 {
		t.Fatalf("expected maxKeys=3, got %d", dist.maxKeys)
	}
	if dist.concurrency != 5 {
		t.Fatalf("expected concurrency=5, got %d", dist.concurrency)
	}
	if dist.logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestWithMaxKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := NewCdn(WithClient(mockcf.NewMockCloudFrontClient(ctrl)), WithMaxKeys(7))
	if d.maxKeys != 7 {
		t.Fatalf("expected maxKeys=7, got %d", d.maxKeys)
	}
}

func TestWithMaxKeysPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for maxKeys <= 0")
		}
	}()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	NewCdn(WithClient(mockcf.NewMockCloudFrontClient(ctrl)), WithMaxKeys(0))
}

func TestWithConcurrency_SetsValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	d := NewCdn(WithClient(mockcf.NewMockCloudFrontClient(ctrl)), WithConcurrency(10))
	if d.concurrency != 10 {
		t.Fatalf("expected concurrency=10, got %d", d.concurrency)
	}
}

func TestWithConcurrencyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for concurrency <= 0")
		}
	}()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	NewCdn(WithClient(mockcf.NewMockCloudFrontClient(ctrl)), WithConcurrency(0))
}

func TestWithLogger_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil logger")
		}
	}()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	NewCdn(WithClient(mockcf.NewMockCloudFrontClient(ctrl)), WithLogger(nil))
}

func TestWithClient_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil client")
		}
	}()
	NewCdn(WithClient(nil))
}

// TestCdnAwsConfigLoader_DefaultImpl exercises the real cdnAwsConfigLoader body.
// awsconfig.LoadDefaultConfig does not require credentials and succeeds in CI.
func TestCdnAwsConfigLoader_DefaultImpl(t *testing.T) {
	_, err := cdnAwsConfigLoader()
	if err != nil {
		t.Logf("cdnAwsConfigLoader returned error (acceptable in no-credentials env): %v", err)
	}
}

func TestNewCdnClient_ConfigLoaderError_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when config loader fails")
		}
	}()
	orig := cdnAwsConfigLoader
	cdnAwsConfigLoader = func() (goaws.Config, error) {
		return goaws.Config{}, errors.New("injected config error")
	}
	defer func() { cdnAwsConfigLoader = orig }()
	newCdnClient(&cdnConfig{}) // no client → hits cdnAwsConfigLoader
}

func TestNewCdnClient_CreatesClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mock := mockcf.NewMockCloudFrontClient(ctrl)
	orig := cdnAwsConfigLoader
	cdnAwsConfigLoader = func() (goaws.Config, error) { return goaws.Config{}, nil }
	defer func() { cdnAwsConfigLoader = orig }()
	c := newCdnClient(&cdnConfig{})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	_ = mock
}

// --- HealthCheck ---

func TestHealthCheck_OK(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().
		ListPublicKeys(gomock.Any(), gomock.Any()).
		Return(&cloudfront.ListPublicKeysOutput{}, nil)

	if err := dist.HealthCheck(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealthCheck_Error(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().
		ListPublicKeys(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("network error"))

	if err := dist.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

// --- CreatePublicKey ---

func TestCreatePublicKey_OK(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().
		CreatePublicKey(gomock.Any(), gomock.Any()).
		Return(&cloudfront.CreatePublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{Id: goaws.String("pk-123")},
		}, nil)

	id, err := dist.CreatePublicKey(context.Background(), contracts.CdnKey{Name: "key1", PEM: "pem"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "pk-123" {
		t.Fatalf("expected id=pk-123, got %s", id)
	}
}

func TestCreatePublicKey_Error(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().
		CreatePublicKey(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("api error"))

	if _, err := dist.CreatePublicKey(context.Background(), contracts.CdnKey{Name: "key1", PEM: "pem"}); err == nil {
		t.Fatal("expected error")
	}
}

// --- EnsureKeyGroup: group does not exist ---

func TestEnsureKeyGroup_Creates(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().
		ListKeyGroups(gomock.Any(), gomock.Any()).
		Return(&cloudfront.ListKeyGroupsOutput{KeyGroupList: &cfTypes.KeyGroupList{}}, nil)
	client.EXPECT().
		CreateKeyGroup(gomock.Any(), gomock.Any()).
		Return(&cloudfront.CreateKeyGroupOutput{
			KeyGroup: &cfTypes.KeyGroup{Id: goaws.String("kg-1")},
		}, nil)

	id, err := dist.EnsureKeyGroup(context.Background(), "my-group", "pk-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "kg-1" {
		t.Fatalf("expected kg-1, got %s", id)
	}
}

// --- EnsureKeyGroup: group exists, update ---

func TestEnsureKeyGroup_Updates(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()

	kgSummary := cfTypes.KeyGroupSummary{
		KeyGroup: &cfTypes.KeyGroup{
			Id:             goaws.String("kg-existing"),
			KeyGroupConfig: &cfTypes.KeyGroupConfig{Name: goaws.String("my-group"), Items: []string{"pk-old"}},
		},
	}
	client.EXPECT().
		ListKeyGroups(gomock.Any(), gomock.Any()).
		Return(&cloudfront.ListKeyGroupsOutput{
			KeyGroupList: &cfTypes.KeyGroupList{Items: []cfTypes.KeyGroupSummary{kgSummary}},
		}, nil)
	client.EXPECT().
		GetKeyGroup(gomock.Any(), gomock.Any()).
		Return(&cloudfront.GetKeyGroupOutput{
			KeyGroup: kgSummary.KeyGroup,
			ETag:     goaws.String("etag-1"),
		}, nil)
	// fetchKeyMetas for both keys
	client.EXPECT().
		GetPublicKey(gomock.Any(), gomock.Any()).
		Return(&cloudfront.GetPublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{CreatedTime: goaws.Time(time.Now())},
		}, nil).AnyTimes()
	client.EXPECT().
		UpdateKeyGroup(gomock.Any(), gomock.Any()).
		Return(&cloudfront.UpdateKeyGroupOutput{}, nil)

	id, err := dist.EnsureKeyGroup(context.Background(), "my-group", "pk-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "kg-existing" {
		t.Fatalf("expected kg-existing, got %s", id)
	}
}

// --- VerifyKeyInGroup ---

func TestVerifyKeyInGroup_Found(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()

	kgSummary := cfTypes.KeyGroupSummary{
		KeyGroup: &cfTypes.KeyGroup{
			Id:             goaws.String("kg-1"),
			KeyGroupConfig: &cfTypes.KeyGroupConfig{Name: goaws.String("my-group"), Items: []string{"pk-1"}},
		},
	}
	client.EXPECT().
		ListKeyGroups(gomock.Any(), gomock.Any()).
		Return(&cloudfront.ListKeyGroupsOutput{
			KeyGroupList: &cfTypes.KeyGroupList{Items: []cfTypes.KeyGroupSummary{kgSummary}},
		}, nil)
	client.EXPECT().
		GetKeyGroup(gomock.Any(), gomock.Any()).
		Return(&cloudfront.GetKeyGroupOutput{KeyGroup: kgSummary.KeyGroup}, nil)
	client.EXPECT().
		GetPublicKey(gomock.Any(), gomock.Any()).
		Return(&cloudfront.GetPublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{
				PublicKeyConfig: &cfTypes.PublicKeyConfig{Name: goaws.String("my-key")},
			},
		}, nil)

	found, err := dist.VerifyKeyInGroup(context.Background(), contracts.CdnKey{GroupName: "my-group", Name: "my-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
}

func TestVerifyKeyInGroup_NotFound(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().
		ListKeyGroups(gomock.Any(), gomock.Any()).
		Return(&cloudfront.ListKeyGroupsOutput{KeyGroupList: &cfTypes.KeyGroupList{}}, nil)

	found, err := dist.VerifyKeyInGroup(context.Background(), contracts.CdnKey{GroupName: "missing", Name: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected key not to be found")
	}
}

// --- WithLogger ---

func TestWithLogger_NonNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	logger := slog.Default()
	d := NewCdn(WithClient(mockcf.NewMockCloudFrontClient(ctrl)), WithLogger(logger))
	if d.logger != logger {
		t.Fatal("expected custom logger to be set")
	}
}

// --- EnsureKeyGroup error paths ---

func TestEnsureKeyGroup_ListKeyGroupsError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().ListKeyGroups(gomock.Any(), gomock.Any()).Return(nil, errors.New("list error"))
	_, err := dist.EnsureKeyGroup(context.Background(), "grp", "pk-1")
	if err == nil {
		t.Fatal("expected error from ListKeyGroups")
	}
}

func TestEnsureKeyGroup_UpdateKeyGroupError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	kgSummary := cfTypes.KeyGroupSummary{
		KeyGroup: &cfTypes.KeyGroup{
			Id:             goaws.String("kg-1"),
			KeyGroupConfig: &cfTypes.KeyGroupConfig{Name: goaws.String("grp"), Items: []string{"pk-old"}},
		},
	}
	client.EXPECT().ListKeyGroups(gomock.Any(), gomock.Any()).Return(
		&cloudfront.ListKeyGroupsOutput{
			KeyGroupList: &cfTypes.KeyGroupList{Items: []cfTypes.KeyGroupSummary{kgSummary}},
		}, nil,
	)
	client.EXPECT().GetKeyGroup(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetKeyGroupOutput{KeyGroup: kgSummary.KeyGroup, ETag: goaws.String("e1")}, nil,
	)
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetPublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{CreatedTime: goaws.Time(time.Now())},
		}, nil,
	).AnyTimes()
	client.EXPECT().UpdateKeyGroup(gomock.Any(), gomock.Any()).Return(nil, errors.New("update error"))
	_, err := dist.EnsureKeyGroup(context.Background(), "grp", "pk-new")
	if err == nil {
		t.Fatal("expected error from UpdateKeyGroup")
	}
}

func TestEnsureKeyGroup_CreateKeyGroupError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().ListKeyGroups(gomock.Any(), gomock.Any()).Return(
		&cloudfront.ListKeyGroupsOutput{KeyGroupList: &cfTypes.KeyGroupList{}}, nil,
	)
	client.EXPECT().CreateKeyGroup(gomock.Any(), gomock.Any()).Return(nil, errors.New("create error"))
	_, err := dist.EnsureKeyGroup(context.Background(), "grp", "pk-1")
	if err == nil {
		t.Fatal("expected error from CreateKeyGroup")
	}
}

// --- findKeyGroupByName: nil KeyGroupList ---

func TestFindKeyGroupByName_NilKeyGroupList(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().ListKeyGroups(gomock.Any(), gomock.Any()).Return(
		&cloudfront.ListKeyGroupsOutput{KeyGroupList: nil}, nil,
	)
	kg, err := dist.findKeyGroupByName(context.Background(), "missing")
	if err != nil || kg != nil {
		t.Fatalf("expected nil,nil: got=%v err=%v", kg, err)
	}
}

// --- updateKeyGroup error paths ---

func TestUpdateKeyGroup_GetKeyGroupError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().GetKeyGroup(gomock.Any(), gomock.Any()).Return(nil, errors.New("get kg error"))
	err := dist.updateKeyGroup(context.Background(), "kg-1", "pk-new")
	if err == nil {
		t.Fatal("expected error from GetKeyGroup")
	}
}

func TestUpdateKeyGroup_WithEviction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mockcf.NewMockCloudFrontClient(ctrl)
	// maxKeys=1 so one key will be evicted
	dist := NewCdn(WithClient(client), WithMaxKeys(1))

	now := time.Now()
	client.EXPECT().GetKeyGroup(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetKeyGroupOutput{
			KeyGroup: &cfTypes.KeyGroup{
				KeyGroupConfig: &cfTypes.KeyGroupConfig{Items: []string{"pk-old"}},
			},
			ETag: goaws.String("e1"),
		}, nil,
	)
	// fetchKeyMetas + deletePublicKey all go through GetPublicKey; use AnyTimes with
	// a response that works for both metadata (CreatedTime) and deletion (ETag).
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetPublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{CreatedTime: goaws.Time(now.Add(-1 * time.Hour))},
			ETag:      goaws.String("etag"),
		}, nil,
	).AnyTimes()
	client.EXPECT().DeletePublicKey(gomock.Any(), gomock.Any()).Return(
		&cloudfront.DeletePublicKeyOutput{}, nil,
	)
	client.EXPECT().UpdateKeyGroup(gomock.Any(), gomock.Any()).Return(
		&cloudfront.UpdateKeyGroupOutput{}, nil,
	)
	if err := dist.updateKeyGroup(context.Background(), "kg-1", "pk-new"); err != nil {
		t.Fatalf("updateKeyGroup with eviction: %v", err)
	}
}

func TestUpdateKeyGroup_EvictionDeleteError_Warns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mockcf.NewMockCloudFrontClient(ctrl)
	dist := NewCdn(WithClient(client), WithMaxKeys(1))

	now := time.Now()
	client.EXPECT().GetKeyGroup(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetKeyGroupOutput{
			KeyGroup: &cfTypes.KeyGroup{
				KeyGroupConfig: &cfTypes.KeyGroupConfig{Items: []string{"pk-old"}},
			},
			ETag: goaws.String("e1"),
		}, nil,
	)
	// fetchKeyMetas for all keys (pk-new + pk-old) — returns CreatedTime + ETag for delete
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetPublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{CreatedTime: goaws.Time(now.Add(-1 * time.Hour))},
			ETag:      goaws.String("etag"),
		}, nil,
	).AnyTimes()
	// deletePublicKey fails → logged as Warn, not returned as error
	client.EXPECT().DeletePublicKey(gomock.Any(), gomock.Any()).Return(nil, errors.New("delete failed"))
	client.EXPECT().UpdateKeyGroup(gomock.Any(), gomock.Any()).Return(
		&cloudfront.UpdateKeyGroupOutput{}, nil,
	)
	if err := dist.updateKeyGroup(context.Background(), "kg-1", "pk-new"); err != nil {
		t.Fatalf("expected success even with delete error (warn only): %v", err)
	}
}

// --- deletePublicKey ---

func TestDeletePublicKey_GetPublicKeyError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(nil, errors.New("get pk error"))
	err := dist.deletePublicKey(context.Background(), "pk-1")
	if err == nil {
		t.Fatal("expected error from GetPublicKey")
	}
}

func TestDeletePublicKey_DeleteError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetPublicKeyOutput{
			PublicKey: &cfTypes.PublicKey{}, ETag: goaws.String("etag"),
		}, nil,
	)
	client.EXPECT().DeletePublicKey(gomock.Any(), gomock.Any()).Return(nil, errors.New("delete error"))
	if err := dist.deletePublicKey(context.Background(), "pk-1"); err == nil {
		t.Fatal("expected error from DeletePublicKey")
	}
}

// --- partitionKeysByAge: fetchKeyMetas error path ---

func TestPartitionKeysByAge_FetchMetasError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	// GetPublicKey fails → fetchKeyMetas returns error → partitionKeysByAge keeps all
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(nil, errors.New("meta error"))
	keep, evict := dist.partitionKeysByAge(context.Background(), []string{"pk-1"})
	if len(keep) != 1 || keep[0] != "pk-1" {
		t.Fatalf("expected all keys kept on error: keep=%v", keep)
	}
	if len(evict) != 0 {
		t.Fatalf("expected no evictions on error: evict=%v", evict)
	}
}

// --- fetchKeyMetas: empty ids ---

func TestFetchKeyMetas_EmptyIDs(t *testing.T) {
	ctrl, _, dist := newMock(t)
	defer ctrl.Finish()
	metas, err := dist.fetchKeyMetas(context.Background(), nil)
	if err != nil || metas != nil {
		t.Fatalf("expected nil,nil for empty ids: metas=%v err=%v", metas, err)
	}
}

// --- findPublicKeyIDByName: GetKeyGroup error ---

func TestFindPublicKeyIDByName_GetKeyGroupError(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	kgSummary := cfTypes.KeyGroupSummary{
		KeyGroup: &cfTypes.KeyGroup{
			Id:             goaws.String("kg-1"),
			KeyGroupConfig: &cfTypes.KeyGroupConfig{Name: goaws.String("grp"), Items: []string{"pk-1"}},
		},
	}
	client.EXPECT().ListKeyGroups(gomock.Any(), gomock.Any()).Return(
		&cloudfront.ListKeyGroupsOutput{
			KeyGroupList: &cfTypes.KeyGroupList{Items: []cfTypes.KeyGroupSummary{kgSummary}},
		}, nil,
	)
	client.EXPECT().GetKeyGroup(gomock.Any(), gomock.Any()).Return(nil, errors.New("get kg fail"))
	_, err := dist.findPublicKeyIDByName(context.Background(), "grp", "my-key")
	if err == nil {
		t.Fatal("expected error from GetKeyGroup")
	}
}

func TestFindPublicKeyIDByName_GetPublicKeyWarn(t *testing.T) {
	ctrl, client, dist := newMock(t)
	defer ctrl.Finish()
	kgSummary := cfTypes.KeyGroupSummary{
		KeyGroup: &cfTypes.KeyGroup{
			Id:             goaws.String("kg-1"),
			KeyGroupConfig: &cfTypes.KeyGroupConfig{Name: goaws.String("grp"), Items: []string{"pk-1"}},
		},
	}
	client.EXPECT().ListKeyGroups(gomock.Any(), gomock.Any()).Return(
		&cloudfront.ListKeyGroupsOutput{
			KeyGroupList: &cfTypes.KeyGroupList{Items: []cfTypes.KeyGroupSummary{kgSummary}},
		}, nil,
	)
	client.EXPECT().GetKeyGroup(gomock.Any(), gomock.Any()).Return(
		&cloudfront.GetKeyGroupOutput{KeyGroup: kgSummary.KeyGroup}, nil,
	)
	// GetPublicKey fails → warn and continue → returns "" with no error
	client.EXPECT().GetPublicKey(gomock.Any(), gomock.Any()).Return(nil, errors.New("pk fail"))
	id, err := dist.findPublicKeyIDByName(context.Background(), "grp", "my-key")
	if err != nil || id != "" {
		t.Fatalf("expected empty id and nil error on pk warn: id=%s err=%v", id, err)
	}
}

// --- pure functions ---

func TestDeduplicateKeys(t *testing.T) {
	result := deduplicateKeys([]string{"a", "b", "a"}, "c")
	if len(result) != 3 || result[0] != "c" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestDeduplicateKeys_NewAlreadyExists(t *testing.T) {
	result := deduplicateKeys([]string{"a", "b"}, "a")
	if len(result) != 2 {
		t.Fatalf("expected 2 unique keys, got %v", result)
	}
}

func TestPartitionByAge(t *testing.T) {
	now := time.Now()
	metas := []keyMeta{
		{id: "old", createdAt: now.Add(-2 * time.Hour)},
		{id: "mid", createdAt: now.Add(-1 * time.Hour)},
		{id: "new", createdAt: now},
	}
	keep, evict := partitionByAge(metas, 2)
	if len(keep) != 2 || keep[0] != "new" || keep[1] != "mid" {
		t.Fatalf("unexpected keep: %v", keep)
	}
	if len(evict) != 1 || evict[0] != "old" {
		t.Fatalf("unexpected evict: %v", evict)
	}
}
