package docli

import (
	"flag"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func TestDropletActionsChangeKernel(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.Int(ArgKernelID, 2, "kernel-id")

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionChangeKernel(c)
	})

}
func TestDropletActionsDisableBackups(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			DisableBackupsFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("DisableBackupsFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionDisableBackups(c)
	})

}
func TestDropletActionsEnableIPv6(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			EnableIPv6Fn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("EnableIPv6Fn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionEnableIPv6(c)
	})

}
func TestDropletActionsEnablePrivateNetworking(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			EnablePrivateNetworkingFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("EnablePrivateNetworkingFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionEnablePrivateNetworking(c)
	})
}
func TestDropletActionsGet(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.Int(ArgActionID, 2, ArgActionID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionGet(c)
	})
}

func TestDropletActionsPasswordReset(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			PasswordResetFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PasswordResetFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionPasswordReset(c)
	})
}

func TestDropletActionsPowerCycle(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			PowerCycleFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerCycleFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionPowerCycle(c)
	})
}
func TestDropletActionsPowerOff(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			PowerOffFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerOffFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionPowerOff(c)
	})
}
func TestDropletActionsPowerOn(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			PowerOnFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerOnFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionPowerOn(c)
	})
}
func TestDropletActionsReboot(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			RebootFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RebootFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionReboot(c)
	})
}

func TestDropletActionsRebuildByImageID(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.String(ArgImage, "2", ArgImageID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionRebuild(c)
	})
}

func TestDropletActionsRebuildByImageSlug(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.String(ArgImage, "slug", "slug")

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionRebuild(c)
	})
}
func TestDropletActionsRename(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.String(ArgDropletName, "name", "name")

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionRename(c)
	})
}
func TestDropletActionsResize(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			ResizeFn: func(id int, slug string, resize bool) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ResizeFn() id = %d; expected %d", got, expected)
				}
				if got, expected := slug, "slug"; got != expected {
					t.Errorf("ResizeFn() name = %q; expected %q", got, expected)
				}
				if got, expected := resize, true; got != expected {
					t.Errorf("ResizeFn() resize = %t; expected %t", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.String(ArgImageSlug, "slug", "image-slug")
	fs.Bool(ArgResizeDisk, true, "resize-disk")

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionResize(c)
	})
}

func TestDropletActionsRestore(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.Int(ArgImageID, 2, ArgImageID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionRestore(c)
	})
}
func TestDropletActionsShutdown(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			ShutdownFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ShutdownFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionShutdown(c)
	})
}
func TestDropletActionsSnapshot(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
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

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)
	fs.String(ArgSnapshotName, "name", "name")

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionSnapshot(c)
	})
}
func TestDropletActionsUpgrade(t *testing.T) {
	client := &godo.Client{
		DropletActions: &DropletActionsServiceMock{
			UpgradeFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RenameFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(ArgDropletID, 1, ArgDropletID)

	WithinTest(cs, fs, func(c *cli.Context) {
		DropletActionUpgrade(c)
	})
}
