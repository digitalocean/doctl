package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestAction_DropletsServiceOpImplementsActionService(t *testing.T) {
	if !Implements((*DropletsService)(nil), new(DropletsServiceOp)) {
		t.Error("DropletsServiceOp does not implement DropletsService")
	}
}

func TestDroplets_ListDroplets(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/droplets", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"droplets": [{"id":1},{"id":2}]}`)
	})

	droplets, _, err := client.Droplet.List()
	if err != nil {
		t.Errorf("Droplets.List returned error: %v", err)
	}

	expected := []Droplet{{ID: 1}, {ID: 2}}
	if !reflect.DeepEqual(droplets, expected) {
		t.Errorf("Droplets.List returned %+v, expected %+v", droplets, expected)
	}
}

func TestDroplets_GetDroplet(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/droplets/12345", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"droplet":{"id":12345}}`)
	})

	droplets, _, err := client.Droplet.Get(12345)
	if err != nil {
		t.Errorf("Droplet.Get returned error: %v", err)
	}

	expected := &DropletRoot{Droplet: &Droplet{ID: 12345}}
	if !reflect.DeepEqual(droplets, expected) {
		t.Errorf("Droplets.Get returned %+v, expected %+v", droplets, expected)
	}
}

func TestDroplets_Create(t *testing.T) {
	setup()
	defer teardown()

	createRequest := &DropletCreateRequest{
		Name:   "name",
		Region: "region",
		Size:   "size",
		Image:  "1",
	}

	mux.HandleFunc("/v2/droplets", func(w http.ResponseWriter, r *http.Request) {
		v := new(DropletCreateRequest)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "POST")
		if !reflect.DeepEqual(v, createRequest) {
			t.Errorf("Request body = %+v, expected %+v", v, createRequest)
		}

		fmt.Fprintf(w, `{"droplet":{"id":1}}`)
	})

	droplet, _, err := client.Droplet.Create(createRequest)
	if err != nil {
		t.Errorf("Droplets.Create returned error: %v", err)
	}

	expected := &DropletRoot{Droplet: &Droplet{ID: 1}}
	if !reflect.DeepEqual(droplet, expected) {
		t.Errorf("Droplets.Create returned %+v, expected %+v", droplet, expected)
	}
}

func TestDroplets_Destroy(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v2/droplets/12345", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Droplet.Delete(12345)
	if err != nil {
		t.Errorf("Droplet.Delete returned error: %v", err)
	}
}

func TestLinks_Actions(t *testing.T) {
	setup()
	defer teardown()

	aLink := Link{ID: 1, Rel: "a", HREF: "http://example.com/a"}

	links := Links{
		Actions: []Link{
			aLink,
			Link{ID: 2, Rel: "b", HREF: "http://example.com/b"},
			Link{ID: 2, Rel: "c", HREF: "http://example.com/c"},
		},
	}

	link := links.Action("a")

	if *link != aLink {
		t.Errorf("expected %+v, got %+v", aLink, link)
	}

}

func TestNetwork_String(t *testing.T) {
	network := &Network{
		IPAddress: "192.168.1.2",
		Netmask:   "255.255.255.0",
		Gateway:   "192.168.1.1",
	}

	stringified := network.String()
	expected := `godo.Network{IPAddress:"192.168.1.2", Netmask:"255.255.255.0", Gateway:"192.168.1.1", Type:""}`
	if expected != stringified {
		t.Errorf("Distribution.String returned %+v, expected %+v", stringified, expected)
	}

}

func TestDroplet_String(t *testing.T) {

	region := &Region{
		Slug:      "region",
		Name:      "Region",
		Sizes:     []string{"1", "2"},
		Available: true,
	}

	image := &Image{
		ID:           1,
		Name:         "Image",
		Distribution: "Ubuntu",
		Slug:         "image",
		Public:       true,
		Regions:      []string{"one", "two"},
	}

	size := &Size{
		Slug:         "size",
		PriceMonthly: 123,
		PriceHourly:  456,
		Regions:      []string{"1", "2"},
	}
	network := &Network{
		IPAddress: "192.168.1.2",
		Netmask:   "255.255.255.0",
		Gateway:   "192.168.1.1",
	}
	networks := &Networks{
		V4: []Network{*network},
	}

	droplet := &Droplet{
		ID:          1,
		Name:        "droplet",
		Memory:      123,
		Vcpus:       456,
		Disk:        789,
		Region:      region,
		Image:       image,
		Size:        size,
		BackupIDs:   []int{1},
		SnapshotIDs: []int{1},
		ActionIDs:   []int{1},
		Locked:      false,
		Status:      "active",
		Networks:    networks,
	}

	stringified := droplet.String()
	expected := `godo.Droplet{ID:1, Name:"droplet", Memory:123, Vcpus:456, Disk:789, Region:godo.Region{Slug:"region", Name:"Region", Sizes:["1" "2"], Available:true}, Image:godo.Image{ID:1, Name:"Image", Distribution:"Ubuntu", Slug:"image", Public:true, Regions:["one" "two"]}, Size:godo.Size{Slug:"size", Memory:0, Vcpus:0, Disk:0, PriceMonthly:123, PriceHourly:456, Regions:["1" "2"]}, BackupIDs:[1], SnapshotIDs:[1], Locked:false, Status:"active", Networks:godo.Networks{V4:[godo.Network{IPAddress:"192.168.1.2", Netmask:"255.255.255.0", Gateway:"192.168.1.1", Type:""}]}, ActionIDs:[1]}`
	if expected != stringified {
		t.Errorf("Droplet.String returned %+v, expected %+v", stringified, expected)
	}
}
