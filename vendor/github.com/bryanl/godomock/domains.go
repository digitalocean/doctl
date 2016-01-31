package godomock

import "github.com/digitalocean/godo"

// DomainsService is the godo DOmainsService interface.
type DomainsService interface {
	List(*godo.ListOptions) ([]godo.Domain, *godo.Response, error)
	Get(string) (*godo.Domain, *godo.Response, error)
	Create(*godo.DomainCreateRequest) (*godo.Domain, *godo.Response, error)
	Delete(string) (*godo.Response, error)

	Records(string, *godo.ListOptions) ([]godo.DomainRecord, *godo.Response, error)
	Record(string, int) (*godo.DomainRecord, *godo.Response, error)
	DeleteRecord(string, int) (*godo.Response, error)
	EditRecord(string, int, *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error)
	CreateRecord(string, *godo.DomainRecordEditRequest) (*godo.DomainRecord, *godo.Response, error)
}
