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
	out := []map[string]interface{}{}

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
		"UpdatedAt",
	}
}

func (r *Repository) ColMap() map[string]string {
	return map[string]string{
		"Name":      "Name",
		"LatestTag": "Latest Tag",
		"UpdatedAt": "Updated At",
	}
}

func (r *Repository) KV() []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, reg := range r.Repositories {
		m := map[string]interface{}{
			"Name":      reg.Name,
			"LatestTag": reg.LatestTag.Tag,
			"UpdatedAt": reg.LatestTag.UpdatedAt,
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
	out := []map[string]interface{}{}

	for _, tag := range r.Tags {
		m := map[string]interface{}{
			"Tag":                 tag.Tag,
			"CompressedSizeBytes": BytesToHumanReadibleUnit(tag.CompressedSizeBytes),
			"UpdatedAt":           tag.UpdatedAt,
			"ManifestDigest":      tag.ManifestDigest,
		}

		out = append(out, m)
	}

	return out
}
