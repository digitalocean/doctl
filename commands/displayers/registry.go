/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package displayers

import (
	"fmt"
	"io"

	"github.com/digitalocean/doctl/do"
)

type Registry struct {
	Registries []do.Registry
}

var _ Displayable = &Registry{}

func (r *Registry) JSON(out io.Writer) error {
	return writeJSON(r.Registries, out)
}

func (r *Registry) Cols() []string {
	return []string{
		"Name",
		"Endpoint",
	}
}

func (r *Registry) ColMap() map[string]string {
	return map[string]string{
		"Name":     "Name",
		"Endpoint": "Endpoint",
	}
}

func (r *Registry) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.Registries))

	for _, reg := range r.Registries {
		m := map[string]interface{}{
			"Name":     reg.Name,
			"Endpoint": reg.Endpoint(),
		}

		out = append(out, m)
	}

	return out
}

type Repository struct {
	Repositories []do.Repository
}

var _ Displayable = &Repository{}

func (r *Repository) JSON(out io.Writer) error {
	return writeJSON(r.Repositories, out)
}

func (r *Repository) Cols() []string {
	return []string{
		"Name",
		"LatestTag",
		"TagCount",
		"UpdatedAt",
	}
}

func (r *Repository) ColMap() map[string]string {
	return map[string]string{
		"Name":      "Name",
		"LatestTag": "Latest Tag",
		"TagCount":  "Tag Count",
		"UpdatedAt": "Updated At",
	}
}

func (r *Repository) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.Repositories))

	for _, reg := range r.Repositories {
		m := map[string]interface{}{
			"Name":      reg.Name,
			"LatestTag": reg.LatestTag.Tag,
			"TagCount":  reg.TagCount,
			"UpdatedAt": reg.LatestTag.UpdatedAt,
		}

		out = append(out, m)
	}

	return out
}

type RepositoryV2 struct {
	Repositories []do.RepositoryV2
}

var _ Displayable = &Repository{}

func (r *RepositoryV2) JSON(out io.Writer) error {
	return writeJSON(r.Repositories, out)
}

func (r *RepositoryV2) Cols() []string {
	return []string{
		"Name",
		"LatestManifest",
		"LatestTag",
		"TagCount",
		"ManifestCount",
		"UpdatedAt",
	}
}

func (r *RepositoryV2) ColMap() map[string]string {
	return map[string]string{
		"Name":           "Name",
		"LatestManifest": "Latest Manifest",
		"LatestTag":      "Latest Tag",
		"TagCount":       "Tag Count",
		"ManifestCount":  "Manifest Count",
		"UpdatedAt":      "Updated At",
	}
}

func (r *RepositoryV2) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.Repositories))

	for _, reg := range r.Repositories {
		latestTag := "<none>" // default when latest manifest has no tags
		if len(reg.LatestManifest.Tags) > 0 {
			latestTag = reg.LatestManifest.Tags[0]
		}
		m := map[string]interface{}{
			"Name":           reg.Name,
			"LatestManifest": reg.LatestManifest.Digest,
			"LatestTag":      latestTag,
			"TagCount":       reg.TagCount,
			"ManifestCount":  reg.ManifestCount,
			"UpdatedAt":      reg.LatestManifest.UpdatedAt,
		}

		out = append(out, m)
	}

	return out
}

type RepositoryTag struct {
	Tags []do.RepositoryTag
}

var _ Displayable = &RepositoryTag{}

func (r *RepositoryTag) JSON(out io.Writer) error {
	return writeJSON(r.Tags, out)
}

func (r *RepositoryTag) Cols() []string {
	return []string{
		"Tag",
		"CompressedSizeBytes",
		"UpdatedAt",
		"ManifestDigest",
	}
}

func (r *RepositoryTag) ColMap() map[string]string {
	return map[string]string{
		"Tag":                 "Tag",
		"CompressedSizeBytes": "Compressed Size",
		"UpdatedAt":           "Updated At",
		"ManifestDigest":      "Manifest Digest",
	}
}

func (r *RepositoryTag) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.Tags))

	for _, tag := range r.Tags {
		m := map[string]interface{}{
			"Tag":                 tag.Tag,
			"CompressedSizeBytes": BytesToHumanReadableUnit(tag.CompressedSizeBytes),
			"UpdatedAt":           tag.UpdatedAt,
			"ManifestDigest":      tag.ManifestDigest,
		}

		out = append(out, m)
	}

	return out
}

type RepositoryManifest struct {
	Manifests []do.RepositoryManifest
}

var _ Displayable = &RepositoryManifest{}

func (r *RepositoryManifest) JSON(out io.Writer) error {
	return writeJSON(r.Manifests, out)
}

func (r *RepositoryManifest) Cols() []string {
	return []string{
		"Digest",
		"CompressedSizeBytes",
		"SizeBytes",
		"UpdatedAt",
		"Tags",
	}
}

func (r *RepositoryManifest) ColMap() map[string]string {
	return map[string]string{
		"Digest":              "Manifest Digest",
		"CompressedSizeBytes": "Compressed Size",
		"SizeBytes":           "Uncompressed Size",
		"UpdatedAt":           "Updated At",
		"Tags":                "Tags",
	}
}

func (r *RepositoryManifest) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.Manifests))

	for _, manifest := range r.Manifests {
		m := map[string]interface{}{
			"Digest":              manifest.Digest,
			"CompressedSizeBytes": BytesToHumanReadableUnit(manifest.CompressedSizeBytes),
			"SizeBytes":           BytesToHumanReadableUnit(manifest.SizeBytes),
			"UpdatedAt":           manifest.UpdatedAt,
			"Tags":                manifest.Tags,
		}

		out = append(out, m)
	}

	return out
}

type GarbageCollection struct {
	GarbageCollections []do.GarbageCollection
}

var _ Displayable = &GarbageCollection{}

func (g *GarbageCollection) JSON(out io.Writer) error {
	return writeJSON(g.GarbageCollections, out)
}

func (g *GarbageCollection) Cols() []string {
	return []string{
		"UUID",
		"RegistryName",
		"Status",
		"CreatedAt",
		"UpdatedAt",
		"BlobsDeleted",
		"FreedBytes",
	}
}

func (g *GarbageCollection) ColMap() map[string]string {
	return map[string]string{
		"UUID":         "UUID",
		"RegistryName": "Registry Name",
		"Status":       "Status",
		"CreatedAt":    "Created At",
		"UpdatedAt":    "Updated At",
		"BlobsDeleted": "Blobs Deleted",
		"FreedBytes":   "Bytes Freed",
	}
}

func (g *GarbageCollection) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(g.GarbageCollections))

	for _, gc := range g.GarbageCollections {
		out = append(out, map[string]interface{}{
			"UUID":         gc.UUID,
			"RegistryName": gc.RegistryName,
			"Status":       gc.Status,
			"CreatedAt":    gc.CreatedAt,
			"UpdatedAt":    gc.UpdatedAt,
			"BlobsDeleted": gc.BlobsDeleted,
			"FreedBytes":   gc.FreedBytes,
		})
	}

	return out
}

type RegistrySubscriptionTiers struct {
	SubscriptionTiers []do.RegistrySubscriptionTier
}

func (t *RegistrySubscriptionTiers) JSON(out io.Writer) error {
	return writeJSON(t, out)
}

func (t *RegistrySubscriptionTiers) Cols() []string {
	return []string{
		"Name",
		"Slug",
		"IncludedRepositories",
		"IncludedStorageBytes",
		"AllowStorageOverage",
		"IncludedBandwidthBytes",
		"MonthlyPriceInCents",
		"Eligible",
		"EligibilityReasons",
	}
}

func (t *RegistrySubscriptionTiers) ColMap() map[string]string {
	return map[string]string{
		"Name":                   "Name",
		"Slug":                   "Slug",
		"IncludedRepositories":   "Included Repositories",
		"IncludedStorageBytes":   "Included Storage",
		"AllowStorageOverage":    "Storage Overage Allowed",
		"IncludedBandwidthBytes": "Included Bandwidth",
		"MonthlyPriceInCents":    "Monthly Price",
		"Eligible":               "Eligible?",
		"EligibilityReasons":     "Not Eligible Because",
	}
}

func (t *RegistrySubscriptionTiers) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(t.SubscriptionTiers))

	for _, tier := range t.SubscriptionTiers {
		out = append(out, map[string]interface{}{
			"Name":                   tier.Name,
			"Slug":                   tier.Slug,
			"IncludedRepositories":   tier.IncludedRepositories,
			"IncludedStorageBytes":   BytesToHumanReadableUnit(tier.IncludedStorageBytes),
			"AllowStorageOverage":    tier.AllowStorageOverage,
			"IncludedBandwidthBytes": BytesToHumanReadableUnit(tier.IncludedBandwidthBytes),
			"MonthlyPriceInCents":    fmt.Sprintf("$%d", tier.MonthlyPriceInCents/100),
			"Eligible":               tier.Eligible,
			"EligibilityReasons":     tier.EligibilityReasons,
		})
	}

	return out
}
