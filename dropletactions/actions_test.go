package dropletactions

import (
	"flag"
	"testing"

	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

var (
	testAction = godo.Action{ID: 1}
)

func TestDropletActionsChangeKernel(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.Int(argKernelID, 2, "kernel-id")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		ChangeKernel(c)
	})

}
func TestDropletActionsDisableBackups(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			DisableBackupsFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("DisableBackupsFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		DisableBackups(c)
	})

}
func TestDropletActionsEnableIPv6(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			EnableIPv6Fn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("EnableIPv6Fn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		EnableIPv6(c)
	})

}
func TestDropletActionsEnablePrivateNetworking(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			EnablePrivateNetworkingFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("EnablePrivateNetworkingFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		EnablePrivateNetworking(c)
	})
}
func TestDropletActionsGet(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.Int(argActionID, 2, argActionID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestDropletActionsPasswordReset(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			PasswordResetFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PasswordResetFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		PasswordReset(c)
	})
}

func TestDropletActionsPowerCycle(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			PowerCycleFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerCycleFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		PowerCycle(c)
	})
}
func TestDropletActionsPowerOff(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			PowerOffFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerOffFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		PowerOff(c)
	})
}
func TestDropletActionsPowerOn(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			PowerOnFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("PowerOnFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		PowerOn(c)
	})
}
func TestDropletActionsReboot(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			RebootFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RebootFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Reboot(c)
	})
}

func TestDropletActionsRebuildByImageID(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.String(argImage, "2", argImageID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Rebuild(c)
	})
}

func TestDropletActionsRebuildByImageSlug(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.String(argImage, "slug", "slug")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Rebuild(c)
	})
}
func TestDropletActionsRename(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.String(argDropletName, "name", "name")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Rename(c)
	})
}
func TestDropletActionsResize(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.String(argImageSlug, "slug", "image-slug")
	fs.Bool(argResizeDisk, true, "resize-disk")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Resize(c)
	})
}

func TestDropletActionsRestore(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.Int(argImageID, 2, argImageID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Restore(c)
	})
}
func TestDropletActionsShutdown(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			ShutdownFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("ShutdownFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Shutdown(c)
	})
}
func TestDropletActionsSnapshot(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)
	fs.String(argSnapshotName, "name", "name")

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Snapshot(c)
	})
}
func TestDropletActionsUpgrade(t *testing.T) {
	client := &godo.Client{
		DropletActions: &docli.DropletActionsServiceMock{
			UpgradeFn: func(id int) (*godo.Action, *godo.Response, error) {
				if got, expected := id, 1; got != expected {
					t.Errorf("RenameFn() id = %d; expected %d", got, expected)
				}
				return &testAction, nil, nil
			},
		},
	}

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.Int(argDropletID, 1, argDropletID)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Upgrade(c)
	})
}
