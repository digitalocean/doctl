/*
Copyright 2026 The Doctl Authors All rights reserved.
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

package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// expectWorkspaceTransferUpload sets mock expectations for a single-part workspace upload.
func expectWorkspaceTransferUpload(t *testing.T, tm *tcMocks, sessionID, workspacePath string, size int64, sha256hex string, isArchive bool, partURL string) {
	t.Helper()
	tm.hostedAgents.EXPECT().
		CreateWorkspaceTransfer(sessionID, gomock.Any()).
		DoAndReturn(func(_ string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
			assert.Equal(t, godo.HostedAgentWorkspaceTransferDirectionUpload, create.Direction)
			assert.Equal(t, workspacePath, create.Path)
			assert.Equal(t, size, create.SizeBytes)
			assert.Equal(t, sha256hex, create.SHA256)
			assert.Equal(t, isArchive, create.IsArchive)
			return &godo.HostedAgentWorkspaceTransfer{
				TransferID: "xfer_1",
				Status:     godo.HostedAgentWorkspaceTransferStatusPending,
				PartSize:   size,
			}, nil
		})
	tm.hostedAgents.EXPECT().
		CreateWorkspaceTransferPartUploadURLs(sessionID, "xfer_1", &godo.HostedAgentWorkspaceTransferPartUploadURLsRequest{PartNumbers: []int{1}}).
		Return(&godo.HostedAgentWorkspaceTransferPartUploadURLs{
			PartURLs: []godo.HostedAgentWorkspaceTransferPartUploadURL{{
				PartNumber: 1,
				UploadURL:  partURL,
			}},
		}, nil)
	tm.hostedAgents.EXPECT().
		CommitWorkspaceTransfer(sessionID, "xfer_1", &godo.HostedAgentWorkspaceTransferCommitRequest{SHA256: sha256hex}).
		Return(&godo.HostedAgentWorkspaceTransfer{
			TransferID: "xfer_1",
			Status:     godo.HostedAgentWorkspaceTransferStatusInProgress,
			SizeBytes:  size,
		}, nil)
	tm.hostedAgents.EXPECT().
		GetWorkspaceTransfer(sessionID, "xfer_1").
		Return(&godo.HostedAgentWorkspaceTransfer{
			TransferID:   "xfer_1",
			Status:       godo.HostedAgentWorkspaceTransferStatusCompleted,
			BytesWritten: size,
		}, nil)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
