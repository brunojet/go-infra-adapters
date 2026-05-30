package aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"golang.org/x/sync/errgroup"

	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

type keyMeta struct {
	id        string
	createdAt time.Time
}

func (d *cdnAdapter) findKeyGroupByName(ctx context.Context, name string) (*cfTypes.KeyGroupSummary, error) {
	out, err := d.client.ListKeyGroups(ctx, &cloudfront.ListKeyGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("list key groups: %w", err)
	}
	if out.KeyGroupList == nil {
		return nil, nil
	}
	for i := range out.KeyGroupList.Items {
		kg := &out.KeyGroupList.Items[i]
		if aws.ToString(kg.KeyGroup.KeyGroupConfig.Name) == name {
			return kg, nil
		}
	}
	return nil, nil
}

func (d *cdnAdapter) createKeyGroup(ctx context.Context, name, keyID string) (string, error) {
	input := &cloudfront.CreateKeyGroupInput{
		KeyGroupConfig: &cfTypes.KeyGroupConfig{
			Name:  aws.String(name),
			Items: []string{keyID},
		},
	}
	out, err := d.client.CreateKeyGroup(ctx, input)
	if err != nil {
		return "", fmt.Errorf("create key group: %w", err)
	}
	return aws.ToString(out.KeyGroup.Id), nil
}

func (d *cdnAdapter) updateKeyGroup(ctx context.Context, groupID, newKeyID string) error {
	kg, err := d.client.GetKeyGroup(ctx, &cloudfront.GetKeyGroupInput{Id: aws.String(groupID)})
	if err != nil {
		return fmt.Errorf("get key group: %w", err)
	}
	keyIDs := deduplicateKeys(kg.KeyGroup.KeyGroupConfig.Items, newKeyID)
	keep, evict := d.partitionKeysByAge(ctx, keyIDs)
	for _, id := range evict {
		if err := d.deletePublicKey(ctx, id); err != nil {
			d.logger.Warn("failed to delete orphaned key", "id", id, "err", err)
		}
	}
	input := &cloudfront.UpdateKeyGroupInput{
		Id:      aws.String(groupID),
		IfMatch: kg.ETag,
		KeyGroupConfig: &cfTypes.KeyGroupConfig{
			Name:  kg.KeyGroup.KeyGroupConfig.Name,
			Items: keep,
		},
	}
	if _, err := d.client.UpdateKeyGroup(ctx, input); err != nil {
		return fmt.Errorf("update key group: %w", err)
	}
	d.logger.Info("KeyGroup updated", "id", groupID, "kept", len(keep), "evicted", len(evict))
	return nil
}

func (d *cdnAdapter) findPublicKeyIDByName(ctx context.Context, groupName, keyName string) (string, error) {
	kg, err := d.findKeyGroupByName(ctx, groupName)
	if err != nil || kg == nil {
		return "", err
	}
	kgFull, err := d.client.GetKeyGroup(ctx, &cloudfront.GetKeyGroupInput{Id: kg.KeyGroup.Id})
	if err != nil {
		return "", fmt.Errorf("get key group details: %w", err)
	}
	for _, keyID := range kgFull.KeyGroup.KeyGroupConfig.Items {
		pk, err := d.client.GetPublicKey(ctx, &cloudfront.GetPublicKeyInput{Id: aws.String(keyID)})
		if err != nil {
			d.logger.Warn("failed to get public key", "id", keyID, "err", err)
			continue
		}
		if aws.ToString(pk.PublicKey.PublicKeyConfig.Name) == keyName {
			return keyID, nil
		}
	}
	return "", nil
}

func (d *cdnAdapter) deletePublicKey(ctx context.Context, keyID string) error {
	pk, err := d.client.GetPublicKey(ctx, &cloudfront.GetPublicKeyInput{Id: aws.String(keyID)})
	if err != nil {
		return fmt.Errorf("get public key: %w", err)
	}
	if _, err = d.client.DeletePublicKey(ctx, &cloudfront.DeletePublicKeyInput{
		Id:      aws.String(keyID),
		IfMatch: pk.ETag,
	}); err != nil {
		return fmt.Errorf("delete public key: %w", err)
	}
	d.logger.Info("CloudFront public key deleted", "id", keyID)
	return nil
}

func (d *cdnAdapter) partitionKeysByAge(ctx context.Context, ids []string) (keep, evict []string) {
	metas, err := d.fetchKeyMetas(ctx, ids)
	if err != nil {
		d.logger.Warn("failed to fetch key metadata, keeping all", "err", err)
		return ids, nil
	}
	return partitionByAge(metas, d.maxKeys)
}

func (d *cdnAdapter) fetchKeyMetas(ctx context.Context, ids []string) ([]keyMeta, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	metas := make([]keyMeta, len(ids))
	sem := make(chan struct{}, d.concurrency)
	eg, egCtx := errgroup.WithContext(ctx)
	for i, id := range ids {
		i, id := i, id
		sem <- struct{}{}
		eg.Go(func() error {
			defer func() { <-sem }()
			got, err := d.client.GetPublicKey(egCtx, &cloudfront.GetPublicKeyInput{Id: aws.String(id)})
			if err != nil {
				return fmt.Errorf("fetch metadata for key %s: %w", id, err)
			}
			km := keyMeta{id: id}
			if got.PublicKey != nil && got.PublicKey.CreatedTime != nil {
				km.createdAt = *got.PublicKey.CreatedTime
			}
			metas[i] = km
			return nil
		})
	}
	return metas, eg.Wait()
}

func deduplicateKeys(existing []string, newID string) []string {
	seen := make(map[string]struct{}, len(existing)+1)
	result := make([]string, 0, len(existing)+1)
	result = append(result, newID)
	seen[newID] = struct{}{}
	for _, id := range existing {
		if _, dup := seen[id]; !dup {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func partitionByAge(metas []keyMeta, limit int) (keep, evict []string) {
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].createdAt.After(metas[j].createdAt)
	})
	for i, m := range metas {
		if i < limit {
			keep = append(keep, m.id)
		} else {
			evict = append(evict, m.id)
		}
	}
	return keep, evict
}
