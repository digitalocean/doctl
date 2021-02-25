package commands

import (
	"strconv"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/stretchr/testify/assert"
)

var (
	testFirewall = do.Firewall{
		Firewall: &godo.Firewall{
			Name: "my firewall",
		},
	}

	testFirewallList = do.Firewalls{
		testFirewall,
	}
)

func TestFirewallCommand(t *testing.T) {
	cmd := Firewall()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "create", "update", "list", "list-by-droplet", "delete", "add-droplets", "remove-droplets", "add-tags", "remove-tags", "add-rules", "remove-rules")
}

func TestFirewallGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tm.firewalls.EXPECT().Get(fID).Return(&testFirewall, nil)

		config.Args = append(config.Args, fID)

		err := RunFirewallGet(config)
		assert.NoError(t, err)
	})
}

func TestFirewallCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		firewallCreateRequest := &godo.FirewallRequest{
			Name: "firewall",
			InboundRules: []godo.InboundRule{
				{
					Protocol:  "icmp",
					PortRange: "",
					Sources:   &godo.Sources{},
				},
				{
					Protocol:  "tcp",
					PortRange: "8000-9000",
					Sources: &godo.Sources{
						Addresses: []string{"127.0.0.0", "0::/0", "::/1"},
					},
				},
			},
			Tags:       []string{"backend"},
			DropletIDs: []int{1, 2},
		}
		tm.firewalls.EXPECT().Create(firewallCreateRequest).Return(&testFirewall, nil)

		config.Doit.Set(config.NS, doctl.ArgFirewallName, "firewall")
		config.Doit.Set(config.NS, doctl.ArgTagNames, []string{"backend"})
		config.Doit.Set(config.NS, doctl.ArgDropletIDs, []string{"1", "2"})
		config.Doit.Set(config.NS, doctl.ArgInboundRules, "protocol:icmp protocol:tcp,ports:8000-9000,address:127.0.0.0,address:0::/0,address:::/1")

		err := RunFirewallCreate(config)
		assert.NoError(t, err)
	})
}

func TestFirewallUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		firewallUpdateRequest := &godo.FirewallRequest{
			Name: "firewall",
			InboundRules: []godo.InboundRule{
				{
					Protocol:  "tcp",
					PortRange: "8000-9000",
					Sources: &godo.Sources{
						Addresses: []string{"127.0.0.0"},
					},
				},
			},
			OutboundRules: []godo.OutboundRule{
				{
					Protocol:  "tcp",
					PortRange: "8080",
					Destinations: &godo.Destinations{
						LoadBalancerUIDs: []string{"lb-uuid"},
						KubernetesIDs:    []string{"doks-01"},
						Tags:             []string{"new-droplets"},
					},
				},
				{
					Protocol:  "tcp",
					PortRange: "80",
					Destinations: &godo.Destinations{
						Addresses: []string{"192.168.0.0"},
					},
				},
			},
			DropletIDs: []int{1},
		}
		tm.firewalls.EXPECT().Update(fID, firewallUpdateRequest).Return(&testFirewall, nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgFirewallName, "firewall")
		config.Doit.Set(config.NS, doctl.ArgDropletIDs, []string{"1"})
		config.Doit.Set(config.NS, doctl.ArgInboundRules, "protocol:tcp,ports:8000-9000,address:127.0.0.0")
		config.Doit.Set(config.NS, doctl.ArgOutboundRules, "protocol:tcp,ports:8080,load_balancer_uid:lb-uuid,kubernetes_id:doks-01,tag:new-droplets protocol:tcp,ports:80,address:192.168.0.0")

		err := RunFirewallUpdate(config)
		assert.NoError(t, err)
	})
}

func TestFirewallList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.firewalls.EXPECT().List().Return(testFirewallList, nil)

		err := RunFirewallList(config)
		assert.NoError(t, err)
	})
}

func TestFirewallListByDroplet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dID := 124
		tm.firewalls.EXPECT().ListByDroplet(dID).Return(testFirewallList, nil)
		config.Args = append(config.Args, strconv.Itoa(dID))

		err := RunFirewallListByDroplet(config)
		assert.NoError(t, err)
	})
}

func TestFirewallDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tm.firewalls.EXPECT().Delete(fID).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunFirewallDelete(config)
		assert.NoError(t, err)
	})
}

func TestFirewallAddDroplets(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		dropletIDs := []int{1, 2}
		tm.firewalls.EXPECT().AddDroplets(fID, dropletIDs[0], dropletIDs[1]).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgDropletIDs, []string{"1", "2"})

		err := RunFirewallAddDroplets(config)
		assert.NoError(t, err)
	})
}

func TestFirewallRemoveDroplets(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "cde2c0d6-41e3-479e-ba60-ad971227232c"
		dropletIDs := []int{1}
		tm.firewalls.EXPECT().RemoveDroplets(fID, dropletIDs[0]).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgDropletIDs, []string{"1"})

		err := RunFirewallRemoveDroplets(config)
		assert.NoError(t, err)
	})
}

func TestFirewallAddTags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tags := []string{"frontend", "backend"}
		tm.firewalls.EXPECT().AddTags(fID, tags[0], tags[1]).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgTagNames, []string{"frontend", "backend"})

		err := RunFirewallAddTags(config)
		assert.NoError(t, err)
	})
}

func TestFirewallRemoveTags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tags := []string{"backend"}
		tm.firewalls.EXPECT().RemoveTags(fID, tags[0]).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgTagNames, []string{"backend"})

		err := RunFirewallRemoveTags(config)
		assert.NoError(t, err)
	})
}

func TestFirewallAddRules(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		inboundRules := []godo.InboundRule{
			{
				Protocol:  "tcp",
				PortRange: "80",
				Sources: &godo.Sources{
					Addresses: []string{"127.0.0.0", "0.0.0.0/0", "2604:A880:0002:00D0:0000:0000:32F1:E001"},
				},
			},
			{
				Protocol:  "tcp",
				PortRange: "8080",
				Sources: &godo.Sources{
					Tags:          []string{"backend"},
					DropletIDs:    []int{1, 2, 3},
					KubernetesIDs: []string{"doks-01"},
				},
			},
		}
		outboundRules := []godo.OutboundRule{
			{
				Protocol:  "tcp",
				PortRange: "22",
				Destinations: &godo.Destinations{
					LoadBalancerUIDs: []string{"lb-uuid"},
					KubernetesIDs:    []string{"doks-02"},
				},
			},
		}
		firewallRulesRequest := &godo.FirewallRulesRequest{
			InboundRules:  inboundRules,
			OutboundRules: outboundRules,
		}

		tm.firewalls.EXPECT().AddRules(fID, firewallRulesRequest).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgInboundRules, "protocol:tcp,ports:80,address:127.0.0.0,address:0.0.0.0/0,address:2604:A880:0002:00D0:0000:0000:32F1:E001 protocol:tcp,ports:8080,tag:backend,droplet_id:1,droplet_id:2,droplet_id:3,kubernetes_id:doks-01")
		config.Doit.Set(config.NS, doctl.ArgOutboundRules, "protocol:tcp,ports:22,load_balancer_uid:lb-uuid,kubernetes_id:doks-02")

		err := RunFirewallAddRules(config)
		assert.NoError(t, err)
	})
}

func TestFirewallRemoveRules(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fID := "ab06e011-6dd1-4034-9293-201f71aba299"
		inboundRules := []godo.InboundRule{
			{
				Protocol:  "tcp",
				PortRange: "80",
				Sources: &godo.Sources{
					Addresses: []string{"0.0.0.0/0"},
				},
			},
		}
		outboundRules := []godo.OutboundRule{
			{
				Protocol:  "tcp",
				PortRange: "22",
				Destinations: &godo.Destinations{
					Tags:      []string{"back:end"},
					Addresses: []string{"::/0"},
				},
			},
		}
		firewallRulesRequest := &godo.FirewallRulesRequest{
			InboundRules:  inboundRules,
			OutboundRules: outboundRules,
		}

		tm.firewalls.EXPECT().RemoveRules(fID, firewallRulesRequest).Return(nil)

		config.Args = append(config.Args, fID)
		config.Doit.Set(config.NS, doctl.ArgInboundRules, "protocol:tcp,ports:80,address:0.0.0.0/0")
		config.Doit.Set(config.NS, doctl.ArgOutboundRules, "protocol:tcp,ports:22,tag:back:end,address:::/0")

		err := RunFirewallRemoveRules(config)
		assert.NoError(t, err)
	})
}
