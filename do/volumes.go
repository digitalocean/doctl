package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// Volume is a wrapper for godo.Volume.
type Volume struct {
	*godo.Volume
}

// VolumesService is an interface for interacting with DigitalOcean's volume api.
type VolumesService interface {
	List() ([]Volume, error)
	CreateVolume(*godo.VolumeCreateRequest) (*Volume, error)
	DeleteVolume(string) error
	Get(string) (*Volume, error)
	CreateSnapshot(*godo.SnapshotCreateRequest) (*Snapshot, error)
}

type volumesService struct {
	client *godo.Client
}

var _ VolumesService = &volumesService{}

// NewVolumesService builds an NewVolumesService instance.
func NewVolumesService(godoClient *godo.Client) VolumesService {
	return &volumesService{
		client: godoClient,
	}
}

func (a *volumesService) List() ([]Volume, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		params := &godo.ListVolumeParams{ListOptions: opt}
		list, resp, err := a.client.Storage.ListVolumes(context.TODO(), params)
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

	list := make([]Volume, len(si))
	for i := range si {
		a := si[i].(godo.Volume)
		list[i] = Volume{Volume: &a}
	}
	return list, nil
}

func (a *volumesService) CreateVolume(r *godo.VolumeCreateRequest) (*Volume, error) {
	al, _, err := a.client.Storage.CreateVolume(context.TODO(), r)
	if err != nil {
		return nil, err
	}
	return &Volume{Volume: al}, nil
}

func (a *volumesService) DeleteVolume(id string) error {
	_, err := a.client.Storage.DeleteVolume(context.TODO(), id)
	if err != nil {
		return err
	}

	return nil
}

func (a *volumesService) Get(id string) (*Volume, error) {
	d, _, err := a.client.Storage.GetVolume(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &Volume{Volume: d}, nil
}

func (a *volumesService) CreateSnapshot(createRequest *godo.SnapshotCreateRequest) (*Snapshot, error) {
	s, _, err := a.client.Storage.CreateSnapshot(context.TODO(), createRequest)
	if err != nil {
		return nil, err
	}

	return &Snapshot{Snapshot: s}, nil
}
