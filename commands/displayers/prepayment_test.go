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
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestPrepaymentConfigKV(t *testing.T) {
	createdAt := time.Date(2026, 6, 22, 19, 31, 51, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 25, 18, 40, 7, 0, time.UTC)

	tests := []struct {
		name              string
		createdAt         time.Time
		updatedAt         time.Time
		expectedCreatedAt string
		expectedUpdatedAt string
	}{
		{
			name:              "zero CreatedAt and UpdatedAt are empty strings",
			createdAt:         time.Time{},
			updatedAt:         time.Time{},
			expectedCreatedAt: "",
			expectedUpdatedAt: "",
		},
		{
			name:              "non-zero CreatedAt and UpdatedAt are RFC3339",
			createdAt:         createdAt,
			updatedAt:         updatedAt,
			expectedCreatedAt: createdAt.Format(time.RFC3339),
			expectedUpdatedAt: updatedAt.Format(time.RFC3339),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PrepaymentConfig{
				PrepaymentConfigResponse: &do.PrepaymentConfigResponse{
					PrepaymentConfigResponse: &godo.PrepaymentConfigResponse{
						Config: &godo.PrepaymentConfig{
							SpendLimit:          "100.00",
							IsAutoPrepayEnabled: true,
							PrepayAmount:        "50.00",
							PrepayThreshold:     "10.00",
							CreatedAt:           tt.createdAt,
							UpdatedAt:           tt.updatedAt,
						},
					},
				},
			}

			kv := p.KV()
			assert.Len(t, kv, 1)
			assert.Equal(t, tt.expectedCreatedAt, kv[0]["CreatedAt"])
			assert.Equal(t, tt.expectedUpdatedAt, kv[0]["UpdatedAt"])
		})
	}
}
