package commands

import (
	"io/ioutil"
	"testing"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

func TestDropletActionCommand(t *testing.T) {
	cmd := DropletAction()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "change-kernel", "disable-backups", "enable-ipv6", "enable-private-networking", "get", "power-cycle", "power-off", "power-on", "power-reset", "reboot", "rebuild", "rename", "resize", "restore", "shutdown", "snapshot", "upgrade")
}

func TestDropletActionsChangeKernel(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			ChangeKernelFn: func(id, kernelID int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ChangeKernelFn() id = %d; expected %d", got, expected)
				}
				if got, expected := kernelID, 2; got != expected {
					t.Errorf("ChangeKernelFn() kernelID = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}

		c.Set(config.ns, doit.ArgKernelID, 2)
		config.args = append(config.args, "1")

		RunDropletActionChangeKernel(config)
	})
}
func TestDropletActionsDisableBackups(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			DisableBackupsFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("DisableBackupsFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionDisableBackups(config)
	})

}
func TestDropletActionsEnableIPv6(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			EnableIPv6Fn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("EnableIPv6Fn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionEnableIPv6(config)
	})
}

func TestDropletActionsEnablePrivateNetworking(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			EnablePrivateNetworkingFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("EnablePrivateNetworkingFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionEnablePrivateNetworking(config)
	})
}
func TestDropletActionsGet(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			GetFn: func(dropletID, actionID int) (*godo.Action, *godo.Response, error) {
				if got, expected := dropletID, 1; got != expected {
					t.Errorf("GetFn() droplet id = %d; expected %d", got, expected)
				}
				if got, expected := actionID, 2; got != expected {
					t.Errorf("GetFn() action id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgActionID, 2)

		RunDropletActionGet(config)
	})
}

func TestDropletActionsPasswordReset(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			PasswordResetFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PasswordResetFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionPasswordReset(config)
	})
}

func TestDropletActionsPowerCycle(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			PowerCycleFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerCycleFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionPowerCycle(config)
	})

}
func TestDropletActionsPowerOff(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			PowerOffFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerOffFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionPowerOff(config)
	})
}
func TestDropletActionsPowerOn(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			PowerOnFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerOnFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionPowerOn(config)
	})

}
func TestDropletActionsReboot(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			RebootFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RebootFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionReboot(config)
	})
}

func TestDropletActionsRebuildByImageID(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			RebuildByImageIDFn: func(id, imageID int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RebuildByImageIDFn() id = %d; expected %d", got, expected)
				}
				if got, expected := imageID, 2; got != expected {
					t.Errorf("RebuildByImageIDFn() image id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgImage, "2")

		RunDropletActionRebuild(config)
	})
}

func TestDropletActionsRebuildByImageSlug(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			RebuildByImageSlugFn: func(id int, slug string) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RebuildByImageSlugFn() id = %d; expected %d", got, expected)
				}
				if got, expected := slug, "slug"; got != expected {
					t.Errorf("RebuildByImageSlugFn() slug = %q; expected %q", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgImage, "slug")

		RunDropletActionRebuild(config)
	})

}
func TestDropletActionsRename(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			RenameFn: func(id int, name string) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RenameFn() id = %d; expected %d", got, expected)
				}
				if got, expected := name, "name"; got != expected {
					t.Errorf("RenameFn() name = %q; expected %q", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgDropletName, "name")

		RunDropletActionRename(config)
	})
}

func TestDropletActionsResize(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			ResizeFn: func(id int, slug string, resize bool) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ResizeFn() id = %d; expected %d", got, expected)
				}
				if got, expected := slug, "1gb"; got != expected {
					t.Errorf("ResizeFn() size = %q; expected %q", got, expected)
				}
				if got, expected := resize, true; got != expected {
					t.Errorf("ResizeFn() resize = %t; expected %t", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgSizeSlug, "1gb")
		c.Set(config.ns, doit.ArgResizeDisk, true)

		RunDropletActionResize(config)
	})
}

func TestDropletActionsRestore(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			RestoreFn: func(id, imageID int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RestoreFn() id = %d; expected %d", got, expected)
				}
				if got, expected := imageID, 2; got != expected {
					t.Errorf("RestoreFn() imageID = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgImageID, 2)

		RunDropletActionRestore(config)
	})
}

func TestDropletActionsShutdown(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			ShutdownFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ShutdownFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionShutdown(config)
	})
}

func TestDropletActionsSnapshot(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			SnapshotFn: func(id int, name string) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ShutdownFn() id = %d; expected %d", got, expected)
				}
				if got, expected := name, "name"; got != expected {
					t.Errorf("ShutdownFn() name = %q; expected %q", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		c.Set(config.ns, doit.ArgSnapshotName, "name")

		RunDropletActionSnapshot(config)
	})
}

func TestDropletActionsUpgrade(t *testing.T) {
	client := &godo.Client{
		DropletActions: &doit.DropletActionsServiceMock{
			UpgradeFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RenameFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	withTestClient(client, func(c *TestConfig) {
		config := &cmdConfig{
			ns:         "test",
			doitConfig: c,
			out:        ioutil.Discard,
		}
		config.args = append(config.args, "1")

		RunDropletActionUpgrade(config)
	})
}
