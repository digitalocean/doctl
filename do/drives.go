package do

import "github.com/digitalocean/godo"

// Drive is a wrapper for godo.Drive.
type Drive struct {
	*godo.Drive
}

// DrivesService is an interface for interacting with DigitalOcean's account api.
type DrivesService interface {
	List() ([]Drive, error)
	CreateDrive(*godo.DriveCreateRequest) (*Drive, error)
	DeleteDrive(string) error
	Get(string) (*Drive, error)
}

type drivesService struct {
	client *godo.Client
}

var _ DrivesService = &drivesService{}

// NewAccountService builds an NewDrivesService instance.
func NewDrivesService(godoClient *godo.Client) DrivesService {
	return &drivesService{
		client: godoClient,
	}

}

func (a *drivesService) List() ([]Drive, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := a.client.Storage.ListDrives(opt)
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

	list := make([]Drive, len(si))
	for i := range si {
		a := si[i].(godo.Drive)
		list[i] = Drive{Drive: &a}

	}

	return list, nil

}

func (a *drivesService) CreateDrive(r *godo.DriveCreateRequest) (*Drive, error) {
	al, _, err := a.client.Storage.CreateDrive(r)
	if err != nil {
		return nil, err

	}
	return &Drive{Drive: al}, nil

}

func (a *drivesService) DeleteDrive(id string) error {

	_, err := a.client.Storage.DeleteDrive(id)
	if err != nil {
		return err

	}

	return nil

}

func (a *drivesService) Get(id string) (*Drive, error) {
	d, _, err := a.client.Storage.GetDrive(id)
	if err != nil {
		return nil, err

	}

	return &Drive{Drive: d}, nil

}
