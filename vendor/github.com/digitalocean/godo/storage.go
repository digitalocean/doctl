package godo

import (
	"fmt"
	"time"
)

const (
	storageBasePath  = "v2"
	storageAllocPath = storageBasePath + "/drives"
	storageSnapPath  = storageBasePath + "/snapshots"
)

// StorageService is an interface for interfacing with the storage
// endpoints of the Digital Ocean API.
// See: https://developers.digitalocean.com/documentation/v2#storage
type StorageService interface {
	ListDrives(*ListOptions) ([]Drive, *Response, error)
	GetDrive(string) (*Drive, *Response, error)
	CreateDrive(*DriveCreateRequest) (*Drive, *Response, error)
	DeleteDrive(string) (*Response, error)

	ListSnapshots(driveID string, opts *ListOptions) ([]Snapshot, *Response, error)
	GetSnapshot(string) (*Snapshot, *Response, error)
	CreateSnapshot(*SnapshotCreateRequest) (*Snapshot, *Response, error)
	DeleteSnapshot(string) (*Response, error)
}

// StorageServiceOp handles communication with the storage drives related methods of the
// DigitalOcean API.
type StorageServiceOp struct {
	client *Client
}

var _ StorageService = &StorageServiceOp{}

// Drive represents a Digital Ocean block store drive.
type Drive struct {
	ID            string    `json:"id"`
	Region        *Region   `json:"region"`
	Name          string    `json:"name"`
	SizeGigaBytes int64     `json:"size_gigabytes"`
	Description   string    `json:"description"`
	DropletIDs    []int     `json:"droplet_ids"`
	CreatedAt     time.Time `json:"created_at"`
}

func (f Drive) String() string {
	return Stringify(f)
}

type storageDrivesRoot struct {
	Drives []Drive `json:"drives"`
	Links  *Links  `json:"links"`
}

type storageDriveRoot struct {
	Drive *Drive `json:"drive"`
	Links *Links `json:"links,omitempty"`
}

// DriveCreateRequest represents a request to create a block store
// drive.
type DriveCreateRequest struct {
	Region        string `json:"region"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SizeGibiBytes int64  `json:"size_gigabytes"`
}

// ListDrives lists all storage drives.
func (svc *StorageServiceOp) ListDrives(opt *ListOptions) ([]Drive, *Response, error) {
	path, err := addOptions(storageAllocPath, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := svc.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(storageDrivesRoot)
	resp, err := svc.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	if l := root.Links; l != nil {
		resp.Links = l
	}

	return root.Drives, resp, nil
}

// CreateDrive creates a storage drive. The name must be unique.
func (svc *StorageServiceOp) CreateDrive(createRequest *DriveCreateRequest) (*Drive, *Response, error) {
	path := storageAllocPath

	req, err := svc.client.NewRequest("POST", path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(storageDriveRoot)
	resp, err := svc.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Drive, resp, nil
}

// GetDrive retrieves an individual storage drive.
func (svc *StorageServiceOp) GetDrive(id string) (*Drive, *Response, error) {
	path := fmt.Sprintf("%s/%s", storageAllocPath, id)

	req, err := svc.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(storageDriveRoot)
	resp, err := svc.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Drive, resp, nil
}

// DeleteDrive deletes a storage drive.
func (svc *StorageServiceOp) DeleteDrive(id string) (*Response, error) {
	path := fmt.Sprintf("%s/%s", storageAllocPath, id)

	req, err := svc.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}
	return svc.client.Do(req, nil)
}

// Snapshot represents a Digital Ocean block store snapshot.
type Snapshot struct {
	ID            string    `json:"id"`
	DriveID       string    `json:"drive_id"`
	Region        *Region   `json:"region"`
	Name          string    `json:"name"`
	SizeGibiBytes int64     `json:"size_gigabytes"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

type storageSnapsRoot struct {
	Snapshots []Snapshot `json:"snapshots"`
	Links     *Links     `json:"links"`
}

type storageSnapRoot struct {
	Snapshot *Snapshot `json:"snapshot"`
	Links    *Links    `json:"links,omitempty"`
}

// SnapshotCreateRequest represents a request to create a block store
// drive.
type SnapshotCreateRequest struct {
	DriveID     string `json:"drive_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListSnapshots lists all snapshots related to a storage drive.
func (svc *StorageServiceOp) ListSnapshots(driveID string, opt *ListOptions) ([]Snapshot, *Response, error) {
	path := fmt.Sprintf("%s/%s/snapshots", storageAllocPath, driveID)
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := svc.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(storageSnapsRoot)
	resp, err := svc.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	if l := root.Links; l != nil {
		resp.Links = l
	}

	return root.Snapshots, resp, nil
}

// CreateSnapshot creates a snapshot of a storage drive.
func (svc *StorageServiceOp) CreateSnapshot(createRequest *SnapshotCreateRequest) (*Snapshot, *Response, error) {
	path := fmt.Sprintf("%s/%s/snapshots", storageAllocPath, createRequest.DriveID)

	req, err := svc.client.NewRequest("POST", path, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(storageSnapRoot)
	resp, err := svc.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Snapshot, resp, nil
}

// GetSnapshot retrieves an individual snapshot.
func (svc *StorageServiceOp) GetSnapshot(id string) (*Snapshot, *Response, error) {
	path := fmt.Sprintf("%s/%s", storageSnapPath, id)

	req, err := svc.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(storageSnapRoot)
	resp, err := svc.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Snapshot, resp, nil
}

// DeleteSnapshot deletes a snapshot.
func (svc *StorageServiceOp) DeleteSnapshot(id string) (*Response, error) {
	path := fmt.Sprintf("%s/%s", storageSnapPath, id)

	req, err := svc.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}
	return svc.client.Do(req, nil)
}
