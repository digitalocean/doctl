package doit

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testKey     = godo.Key{ID: 1, Fingerprint: "fingerprint"}
	testKeyList = []godo.Key{testKey}
)

func TestKeysList(t *testing.T) {
	didList := false

	client := &godo.Client{
		Keys: &KeysServiceMock{
			ListFn: func(opts *godo.ListOptions) ([]godo.Key, *godo.Response, error) {
				didList = true

				resp := &godo.Response{
					Links: &godo.Links{
						Pages: &godo.Pages{},
					},
				}

				return testKeyList, resp, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyList(c)
		assert.True(t, didList)
	})
}

func TestKeysGetByID(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			GetByIDFn: func(id int) (*godo.Key, *godo.Response, error) {
				assert.Equal(t, id, testKey.ID)
				return &testKey, nil, nil
			},
			GetByFingerprintFn: func(_ string) (*godo.Key, *godo.Response, error) {
				t.Error("should not request by fingerprint")
				return nil, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKey, "1", ArgKey)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyGet(c)
	})
}

func TestKeysGetByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			GetByFingerprintFn: func(fingerprint string) (*godo.Key, *godo.Response, error) {
				assert.Equal(t, fingerprint, testKey.Fingerprint)
				return &testKey, nil, nil
			},
			GetByIDFn: func(_ int) (*godo.Key, *godo.Response, error) {
				t.Error("should not request by id")
				return nil, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKey, testKey.Fingerprint, ArgKey)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyGet(c)
	})
}

func TestKeysCreate(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			CreateFn: func(req *godo.KeyCreateRequest) (*godo.Key, *godo.Response, error) {
				expected := &godo.KeyCreateRequest{
					Name:      "the key",
					PublicKey: "fingerprint",
				}
				assert.Equal(t, req, expected)
				return &testKey, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKeyName, "the key", ArgKeyName)
	fs.String(ArgKeyPublicKey, "fingerprint", ArgKeyPublicKey)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyCreate(c)
	})
}

func TestKeysDeleteByID(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			DeleteByIDFn: func(id int) (*godo.Response, error) {
				assert.Equal(t, id, 1)
				return nil, nil
			},
			DeleteByFingerprintFn: func(fingerprint string) (*godo.Response, error) {
				t.Errorf("should not call fingerprint")
				return nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKey, "1", ArgKey)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyDelete(c)
	})
}

func TestKeysDeleteByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			DeleteByFingerprintFn: func(fingerprint string) (*godo.Response, error) {
				assert.Equal(t, fingerprint, "fingerprint")
				return nil, nil
			},
			DeleteByIDFn: func(_ int) (*godo.Response, error) {
				t.Errorf("should not call id")
				return nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKey, "fingerprint", ArgKey)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyDelete(c)
	})
}

func TestKeysUpdateByID(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			UpdateByIDFn: func(id int, req *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
				expected := &godo.KeyUpdateRequest{
					Name: "the key",
				}
				assert.Equal(t, req, expected)
				assert.Equal(t, id, 1)
				return &testKey, nil, nil
			},
			UpdateByFingerprintFn: func(_ string, _ *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
				t.Error("should update by id")
				return nil, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKey, "1", ArgKey)
	fs.String(ArgKeyName, "the key", ArgKeyName)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyUpdate(c)
	})
}

func TestKeysUpdateByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &KeysServiceMock{
			UpdateByFingerprintFn: func(fingerprint string, req *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
				expected := &godo.KeyUpdateRequest{
					Name: "the key",
				}
				assert.Equal(t, req, expected)
				assert.Equal(t, fingerprint, "fingerprint")
				return &testKey, nil, nil
			},
			UpdateByIDFn: func(_ int, _ *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error) {
				t.Error("should update by fingerprint")
				return nil, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKey, "fingerprint", ArgKey)
	fs.String(ArgKeyName, "the key", ArgKeyName)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyUpdate(c)
	})
}

func TestSSHPublicKeyImport(t *testing.T) {
	pubkey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCn6eZ8ve0ha04rPRZuoPXK1AQ/h21qslWCzoDcOciXn5OcyafkZw+31k/afaBTeW62D8fXd8e/1xWbFfp/2GqmslYpNCTPrtpNhsE8I0yKjJ8FxX9FfsCOu/Sv83dWgSpiT7pNWVKarZjW9KdKKRQljq1i+H5pX3r5Q9I1v+66mYTe7qsKGas9KWy0vkGoNSqmTCl+d+Y0286chtqBqBjSCUCI8oLKPnJB86Lj344tFGmzDIsJKXMVHTL0dF8n3u6iWN4qiRU+JvkoIkI3v0JvyZXxhR2uPIS1yUAY2GC+2O5mfxydJQzBdtag5Uw8Y7H5yYR1gar/h16bAy5XzRvp testkey"
	path := filepath.Join(os.TempDir(), "key.pub")
	err := ioutil.WriteFile(path, []byte(pubkey), 0600)
	assert.NoError(t, err)
	defer os.Remove(path)

	client := &godo.Client{
		Keys: &KeysServiceMock{
			CreateFn: func(req *godo.KeyCreateRequest) (*godo.Key, *godo.Response, error) {
				expected := &godo.KeyCreateRequest{
					Name:      "testkey",
					PublicKey: pubkey,
				}
				assert.Equal(t, req, expected)
				return &testKey, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKeyPublicKey, "fingerprint", ArgKeyPublicKey)
	fs.String(ArgKeyPublicKeyFile, path, ArgKeyPublicKeyFile)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyImport(c)
	})
}

func TestSSHPublicKeyImportWithName(t *testing.T) {
	pubkey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCn6eZ8ve0ha04rPRZuoPXK1AQ/h21qslWCzoDcOciXn5OcyafkZw+31k/afaBTeW62D8fXd8e/1xWbFfp/2GqmslYpNCTPrtpNhsE8I0yKjJ8FxX9FfsCOu/Sv83dWgSpiT7pNWVKarZjW9KdKKRQljq1i+H5pX3r5Q9I1v+66mYTe7qsKGas9KWy0vkGoNSqmTCl+d+Y0286chtqBqBjSCUCI8oLKPnJB86Lj344tFGmzDIsJKXMVHTL0dF8n3u6iWN4qiRU+JvkoIkI3v0JvyZXxhR2uPIS1yUAY2GC+2O5mfxydJQzBdtag5Uw8Y7H5yYR1gar/h16bAy5XzRvp testkey"
	path := filepath.Join(os.TempDir(), "key.pub")
	err := ioutil.WriteFile(path, []byte(pubkey), 0600)
	assert.NoError(t, err)
	defer os.Remove(path)

	client := &godo.Client{
		Keys: &KeysServiceMock{
			CreateFn: func(req *godo.KeyCreateRequest) (*godo.Key, *godo.Response, error) {
				expected := &godo.KeyCreateRequest{
					Name:      "custom",
					PublicKey: pubkey,
				}
				assert.Equal(t, req, expected)
				return &testKey, nil, nil
			},
		},
	}

	cs := NewTestConfig(client)
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(ArgKeyName, "custom", ArgKeyName)
	fs.String(ArgKeyPublicKeyFile, path, ArgKeyPublicKeyFile)

	withinTest(cs, fs, func(c *cli.Context) {
		KeyImport(c)
	})
}
