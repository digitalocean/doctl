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

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// Certificate wraps a godo Certificate.
type Certificate struct {
	*godo.Certificate
}

// Certificates is a slice of Certificate.
type Certificates []Certificate

// CertificatesService is the godo CertificatesService interface.
type CertificatesService interface {
	Get(cID string) (*Certificate, error)
	Create(cr *godo.CertificateRequest) (*Certificate, error)
	List() (Certificates, error)
	Delete(cID string) error
}

var _ CertificatesService = &certificatesService{}

type certificatesService struct {
	client *godo.Client
}

// NewCertificatesService builds an instance of CertificatesService.
func NewCertificatesService(client *godo.Client) CertificatesService {
	return &certificatesService{
		client: client,
	}
}

func (cs *certificatesService) Get(cID string) (*Certificate, error) {
	c, _, err := cs.client.Certificates.Get(context.TODO(), cID)
	if err != nil {
		return nil, err
	}

	return &Certificate{Certificate: c}, nil
}

func (cs *certificatesService) Create(cr *godo.CertificateRequest) (*Certificate, error) {
	c, _, err := cs.client.Certificates.Create(context.TODO(), cr)
	if err != nil {
		return nil, err
	}

	return &Certificate{Certificate: c}, nil
}

func (cs *certificatesService) List() (Certificates, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := cs.client.Certificates.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]Certificate, len(si))
	for i := range si {
		a := si[i].(godo.Certificate)
		list[i] = Certificate{Certificate: &a}
	}

	return list, nil
}

func (cs *certificatesService) Delete(cID string) error {
	_, err := cs.client.Certificates.Delete(context.TODO(), cID)
	return err
}
