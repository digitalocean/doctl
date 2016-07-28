package ssh

import (
	"crypto/rsa"
	"reflect"
	"testing"
)

func correctPassword(p string) (string, error) {
	return "changeme", nil
}

func wrongPassword(p string) (string, error) {
	return "wrongpwd", nil
}

func emptyPassword(p string) (string, error) {
	return "", nil
}

func TestParsePrivateKey_keyWithPassword(t *testing.T) {
	path := "./testdata/id_rsa_with_password"
	k, err := parsePrivateKey(path, correctPassword)
	if err != nil {
		t.Fatalf("Couldn't parse private key (%s): %s\n", path, err)
	}
	if _, ok := k.(*rsa.PrivateKey); !ok {
		t.Fatalf("Key type should be *rsa.PrivateKey, but is: %v", reflect.TypeOf(k))
	}
}

func TestParsePrivateKey_keyWithPasswordWrongPassword(t *testing.T) {
	providers := []passwordProvider{
		wrongPassword,
		emptyPassword,
	}
	path := "./testdata/id_rsa_with_password"
	for _, p := range providers {
		k, err := parsePrivateKey(path, p)
		if err == nil {
			pwd, _ := p("")
			t.Fatalf("parsePrivateKey should return an error %s, '%s'", path, pwd)
		}
		if k != nil {
			t.Fatalf("parsePrivateKey shouldn't return a key but did: %v", k)
		}
	}
}

func TestParsePrivateKey_keyWithoutPassword(t *testing.T) {
	path := "./testdata/id_rsa_without_password"
	k, err := parsePrivateKey(path, nil)
	if err != nil {
		t.Fatalf("Couldn't parse private key (%s): %s\n", path, err)
	}
	if _, ok := k.(*rsa.PrivateKey); !ok {
		t.Fatalf("Key type should be *rsa.PrivateKey, but is: %v", reflect.TypeOf(k))
	}
}
