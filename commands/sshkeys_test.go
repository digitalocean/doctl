package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

var (
	testKey     = godo.Key{ID: 1, Fingerprint: "fingerprint"}
	testKeyList = []godo.Key{testKey}
)

func TestKeysList(t *testing.T) {
	didList := false

	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		RunKeyList(ns, c, ioutil.Discard)
		assert.True(t, didList)
	})
}

func TestKeysGetByID(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKey, "1")

		RunKeyGet(ns, c, ioutil.Discard)
	})
}

func TestKeysGetByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKey, testKey.Fingerprint)

		RunKeyGet(ns, c, ioutil.Discard)
	})
}

func TestKeysCreate(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKeyName, "the key")
		c.Set(ns, doit.ArgKeyPublicKey, "fingerprint")

		RunKeyCreate(ns, c, ioutil.Discard)
	})
}

func TestKeysDeleteByID(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKey, "1")

		RunKeyDelete(ns, c, ioutil.Discard)
	})
}

func TestKeysDeleteByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKey, "fingerprint")

		RunKeyDelete(ns, c, ioutil.Discard)
	})

}

func TestKeysUpdateByID(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKey, "1")
		c.Set(ns, doit.ArgKeyName, "the key")

		RunKeyUpdate(ns, c, ioutil.Discard)
	})

}

func TestKeysUpdateByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKey, "fingerprint")
		c.Set(ns, doit.ArgKeyName, "the key")

		RunKeyUpdate(ns, c, ioutil.Discard)
	})

}

func TestSSHPublicKeyImport(t *testing.T) {
	pubkey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCn6eZ8ve0ha04rPRZuoPXK1AQ/h21qslWCzoDcOciXn5OcyafkZw+31k/afaBTeW62D8fXd8e/1xWbFfp/2GqmslYpNCTPrtpNhsE8I0yKjJ8FxX9FfsCOu/Sv83dWgSpiT7pNWVKarZjW9KdKKRQljq1i+H5pX3r5Q9I1v+66mYTe7qsKGas9KWy0vkGoNSqmTCl+d+Y0286chtqBqBjSCUCI8oLKPnJB86Lj344tFGmzDIsJKXMVHTL0dF8n3u6iWN4qiRU+JvkoIkI3v0JvyZXxhR2uPIS1yUAY2GC+2O5mfxydJQzBdtag5Uw8Y7H5yYR1gar/h16bAy5XzRvp testkey"
	path := filepath.Join(os.TempDir(), "key.pub")
	err := ioutil.WriteFile(path, []byte(pubkey), 0600)
	assert.NoError(t, err)
	defer os.Remove(path)

	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKeyPublicKey, "fingerprint")
		c.Set(ns, doit.ArgKeyPublicKeyFile, path)

		RunKeyImport(ns, c, ioutil.Discard)
	})

}

func TestSSHPublicKeyImportWithName(t *testing.T) {
	pubkey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCn6eZ8ve0ha04rPRZuoPXK1AQ/h21qslWCzoDcOciXn5OcyafkZw+31k/afaBTeW62D8fXd8e/1xWbFfp/2GqmslYpNCTPrtpNhsE8I0yKjJ8FxX9FfsCOu/Sv83dWgSpiT7pNWVKarZjW9KdKKRQljq1i+H5pX3r5Q9I1v+66mYTe7qsKGas9KWy0vkGoNSqmTCl+d+Y0286chtqBqBjSCUCI8oLKPnJB86Lj344tFGmzDIsJKXMVHTL0dF8n3u6iWN4qiRU+JvkoIkI3v0JvyZXxhR2uPIS1yUAY2GC+2O5mfxydJQzBdtag5Uw8Y7H5yYR1gar/h16bAy5XzRvp testkey"
	path := filepath.Join(os.TempDir(), "key.pub")
	err := ioutil.WriteFile(path, []byte(pubkey), 0600)
	assert.NoError(t, err)
	defer os.Remove(path)

	client := &godo.Client{
		Keys: &doit.KeysServiceMock{
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

	withTestClient(client, func(c *TestConfig) {
		ns := "test"
		c.Set(ns, doit.ArgKeyName, "custom")
		c.Set(ns, doit.ArgKeyPublicKeyFile, path)

		RunKeyImport(ns, c, ioutil.Discard)
	})
}
