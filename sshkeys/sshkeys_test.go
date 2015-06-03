package sshkeys

import (
	"flag"
	"testing"

	"github.com/bryanl/docli"
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
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		List(c)
	})
}

func TestKeysGetByID(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKey, "1", argKey)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestKeysGetByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKey, testKey.Fingerprint, argKey)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Get(c)
	})
}

func TestKeysCreate(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKeyName, "the key", argKeyName)
	fs.String(argKeyPublicKey, "fingerprint", argKeyPublicKey)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Create(c)
	})
}

func TestKeysDeleteByID(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKey, "1", argKey)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Delete(c)
	})
}

func TestKeysDeleteByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKey, "fingerprint", argKey)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Delete(c)
	})
}

func TestKeysUpdateByID(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKey, "1", argKey)
	fs.String(argKeyName, "the key", argKeyName)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Update(c)
	})
}

func TestKeysUpdateByFingerprint(t *testing.T) {
	client := &godo.Client{
		Keys: &docli.KeysServiceMock{
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

	cs := &docli.TestClientSource{client}
	fs := flag.NewFlagSet("flag set", 0)
	fs.String(argKey, "fingerprint", argKey)
	fs.String(argKeyName, "the key", argKeyName)

	docli.WithinTest(cs, fs, func(c *cli.Context) {
		Update(c)
	})
}
