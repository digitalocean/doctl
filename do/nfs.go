/*
Copyright 2025 The Doctl Authors All rights reserved.
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

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// Nfs wraps a godo.Nfs.
type Nfs struct {
	*godo.Nfs
}

// NfsSnapshot wraps a godo.NfsSnapshot.
type NfsSnapshot struct {
	*godo.NfsSnapshot
}

// NfsAccessPoint wraps an NFS access point payload.
type NfsAccessPoint struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	ShareID      string               `json:"share_id"`
	Path         string               `json:"path"`
	Status       string               `json:"status"`
	AccessPolicy NfsAccessPointPolicy `json:"access_policy"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
	IsDefault    bool                 `json:"is_default"`
	VpcID        *string              `json:"vpc_id,omitempty"`
	VpcIDs       []string             `json:"vpc_ids,omitempty"`
}

// NfsAccessPointPolicy describes access policy settings for an access point.
type NfsAccessPointPolicy struct {
	Anonuid                    uint64   `json:"anonuid"`
	Anongid                    uint64   `json:"anongid"`
	Protocols                  []string `json:"protocols"`
	SquashConfig               string   `json:"squash_config"`
	IdentityEnforcementEnabled bool     `json:"identity_enforcement_enabled"`
}

// NfsAccessPointCreateRequest describes the create access point request payload.
type NfsAccessPointCreateRequest struct {
	Name         string               `json:"name"`
	Path         string               `json:"path"`
	AccessPolicy NfsAccessPointPolicy `json:"access_policy"`
	VpcID        string               `json:"vpc_id"`
}

// NfsAccessPointActionResponse wraps mutation responses for access points.
type NfsAccessPointActionResponse struct {
	AccessPoint *NfsAccessPoint `json:"access_point"`
	Action      *godo.NfsAction `json:"action"`
}

// NfsService is an interface for interacting with DigitalOcean's NFS API.
type NfsService interface {
	List(region string) ([]Nfs, error)
	Create(*godo.NfsCreateRequest) (*Nfs, error)
	Delete(id, region string) error
	Get(id, region string) (*Nfs, error)
	ListSnapshots(shareID, region string) ([]NfsSnapshot, error)
	GetSnapshot(snapshotID, region string) (*NfsSnapshot, error)
	DeleteSnapshot(snapshotID, region string) error
	CreateAccessPoint(shareID string, r *NfsAccessPointCreateRequest) (*NfsAccessPointActionResponse, error)
	GetAccessPoint(accessPointID string) (*NfsAccessPoint, error)
	ListAccessPoints(shareID string, status string) ([]NfsAccessPoint, error)
	DeleteAccessPoint(accessPointID string) (*NfsAccessPointActionResponse, error)
}

type nfsService struct {
	client *godo.Client
}

var _ NfsService = &nfsService{}

// NewNfsService builds a NewNfsService instance.
func NewNfsService(godoClient *godo.Client) NfsService {
	return &nfsService{
		client: godoClient,
	}
}

func (s *nfsService) List(region string) ([]Nfs, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.Nfs.List(context.TODO(), opt, region)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = *list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]Nfs, len(si))
	for i := range si {
		nfs := si[i].(godo.Nfs)
		list[i] = Nfs{Nfs: &nfs}
	}
	return list, nil
}

func (s *nfsService) Create(r *godo.NfsCreateRequest) (*Nfs, error) {
	nfs, _, err := s.client.Nfs.Create(context.TODO(), r)
	if err != nil {
		return nil, err
	}
	return &Nfs{Nfs: nfs}, nil
}

func (s *nfsService) Delete(id, region string) error {
	_, err := s.client.Nfs.Delete(context.TODO(), id, region)
	return err
}

func (s *nfsService) Get(id, region string) (*Nfs, error) {
	nfs, _, err := s.client.Nfs.Get(context.TODO(), id, region)
	if err != nil {
		return nil, err
	}

	return &Nfs{Nfs: nfs}, nil
}

func (s *nfsService) ListSnapshots(shareID, region string) ([]NfsSnapshot, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.Nfs.ListSnapshots(context.TODO(), opt, shareID, region)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = *list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]NfsSnapshot, len(si))
	for i := range si {
		snapshot := si[i].(godo.NfsSnapshot)
		list[i] = NfsSnapshot{NfsSnapshot: &snapshot}
	}
	return list, nil
}

func (s *nfsService) GetSnapshot(snapshotID, region string) (*NfsSnapshot, error) {
	snapshot, _, err := s.client.Nfs.GetSnapshot(context.TODO(), snapshotID, region)
	if err != nil {
		return nil, err
	}

	return &NfsSnapshot{NfsSnapshot: snapshot}, nil
}

func (s *nfsService) DeleteSnapshot(snapshotID, region string) error {
	_, err := s.client.Nfs.DeleteSnapshot(context.TODO(), snapshotID, region)
	return err
}

func (s *nfsService) CreateAccessPoint(shareID string, r *NfsAccessPointCreateRequest) (*NfsAccessPointActionResponse, error) {
	resp, _, err := s.client.Nfs.CreateAccessPoint(context.TODO(), shareID, nfsAccessPointCreateRequestToGodo(r))
	if err != nil {
		return nil, err
	}

	return nfsAccessPointActionResponseFromGodo(resp), nil
}

func (s *nfsService) GetAccessPoint(accessPointID string) (*NfsAccessPoint, error) {
	ap, _, err := s.client.Nfs.GetAccessPoint(context.TODO(), accessPointID)
	if err != nil {
		return nil, err
	}

	return nfsAccessPointFromGodo(ap), nil
}

func (s *nfsService) ListAccessPoints(shareID string, status string) ([]NfsAccessPoint, error) {
	opts := &godo.NfsListAccessPointsOptions{}
	if status != "" {
		opts.Status = godo.NfsAccessPointStatus(status)
	}

	list, _, err := s.client.Nfs.ListAccessPoints(context.TODO(), shareID, opts)
	if err != nil {
		return nil, err
	}

	out := make([]NfsAccessPoint, 0, len(list))
	for _, ap := range list {
		if converted := nfsAccessPointFromGodo(ap); converted != nil {
			out = append(out, *converted)
		}
	}

	return out, nil
}

func (s *nfsService) DeleteAccessPoint(accessPointID string) (*NfsAccessPointActionResponse, error) {
	resp, _, err := s.client.Nfs.DeleteAccessPoint(context.TODO(), accessPointID)
	if err != nil {
		return nil, err
	}

	return nfsAccessPointActionResponseFromGodo(resp), nil
}

func nfsAccessPointCreateRequestToGodo(r *NfsAccessPointCreateRequest) *godo.NfsCreateAccessPointRequest {
	protocols := make([]godo.NfsAccessPolicyProtocol, 0, len(r.AccessPolicy.Protocols))
	for _, p := range r.AccessPolicy.Protocols {
		protocols = append(protocols, godo.NfsAccessPolicyProtocol(p))
	}

	return &godo.NfsCreateAccessPointRequest{
		Name:  r.Name,
		Path:  r.Path,
		VpcID: r.VpcID,
		AccessPolicy: godo.NfsAccessPolicy{
			Anonuid:                    r.AccessPolicy.Anonuid,
			Anongid:                    r.AccessPolicy.Anongid,
			Protocols:                  protocols,
			SquashConfig:               godo.NfsSquashConfig(r.AccessPolicy.SquashConfig),
			IdentityEnforcementEnabled: r.AccessPolicy.IdentityEnforcementEnabled,
		},
	}
}

func nfsAccessPointFromGodo(ap *godo.NfsAccessPoint) *NfsAccessPoint {
	if ap == nil {
		return nil
	}

	protocols := make([]string, 0, len(ap.AccessPolicy.Protocols))
	for _, p := range ap.AccessPolicy.Protocols {
		protocols = append(protocols, string(p))
	}

	return &NfsAccessPoint{
		ID:      ap.ID,
		Name:    ap.Name,
		ShareID: ap.ShareID,
		Path:    ap.Path,
		Status:  string(ap.Status),
		AccessPolicy: NfsAccessPointPolicy{
			Anonuid:                    ap.AccessPolicy.Anonuid,
			Anongid:                    ap.AccessPolicy.Anongid,
			Protocols:                  protocols,
			SquashConfig:               string(ap.AccessPolicy.SquashConfig),
			IdentityEnforcementEnabled: ap.AccessPolicy.IdentityEnforcementEnabled,
		},
		CreatedAt: ap.CreatedAt,
		UpdatedAt: ap.UpdatedAt,
		IsDefault: ap.IsDefault,
		VpcID:     ap.VpcID,
		VpcIDs:    nfsAccessPointVpcIDs(ap),
	}
}

func nfsAccessPointVpcIDs(ap *godo.NfsAccessPoint) []string {
	if len(ap.VpcIDs) > 0 {
		return ap.VpcIDs
	}
	if ap.VpcID != nil && *ap.VpcID != "" {
		return []string{*ap.VpcID}
	}
	return nil
}

func nfsAccessPointActionResponseFromGodo(resp *godo.NfsAccessPointActionResponse) *NfsAccessPointActionResponse {
	if resp == nil {
		return nil
	}

	return &NfsAccessPointActionResponse{
		AccessPoint: nfsAccessPointFromGodo(resp.AccessPoint),
		Action:      resp.Action,
	}
}
