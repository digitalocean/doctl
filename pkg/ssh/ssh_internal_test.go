package ssh

import (
	"encoding/pem"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func loadTestData(t *testing.T, n string) []byte {
	testDataDir := "./testdata"
	p := filepath.Join(testDataDir, n)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		t.Fatalf("Error while trying to read %s: %v\n", n, err)
	}

	return b
}

func TestSignerFromKey_keyWithEmptyPassphrase(t *testing.T) {
	key := loadTestData(t, "id_rsa_without_password")
	s, err := signerFromKey(key)
	if err != nil {
		t.Fatalf("signerFromKey shouldn't return an error: %s\n", err)
	}
	if s == nil {
		t.Fatalf("signerFromKey should return a non-nil signer.\n")
	}
}

func TestSignerFromKey_keyWithPassphrase(t *testing.T) {
	key := loadTestData(t, "id_rsa_with_password")
	s, err := signerFromKey(key)
	if err == nil {
		t.Fatalf("signerFromKey should return an error\n")
	}
	if s != nil {
		t.Fatalf("signerFromKey shouldn't return signer.\n")
	}
}

func TestSignerFromEncryptedKey_keyWithPassphrase(t *testing.T) {
	key := loadTestData(t, "id_rsa_with_password")

	// Convert key to PEM
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		t.Fatalf("An error occured while trying to decode id_rsa_with_password\n")
	}

	s, err := signerFromEncryptedKey(pemBlock, []byte("changeme"))
	if err != nil {
		t.Fatalf("signerFromEncryptedKey shouldn't return an error: %s\n", err)
	}
	if s == nil {
		t.Fatalf("signerFromEncryptedKey should return a non-nil signer.\n")
	}
}

func TestSignerFromEncryptedKey_keyWithPassphraseWrongPassword(t *testing.T) {
	key := loadTestData(t, "id_rsa_with_password")

	// Convert key to PEM
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		t.Fatalf("An error occured while trying to decode id_rsa_with_password\n")
	}

	s, err := signerFromEncryptedKey(pemBlock, []byte("wrongpassword"))
	if err == nil {
		t.Fatalf("signerFromEncryptedKey should return an error.\n")
	}
	if s != nil {
		t.Fatalf("signerFromEncryptedKey shouldn't return a signer.\n")
	}
}

func TestSignerFromEncryptedKey_keyWithEmptyPassphrase(t *testing.T) {
	key := loadTestData(t, "id_rsa_without_password")

	// Convert key to PEM
	pemBlock, _ := pem.Decode(key)
	if pemBlock == nil {
		t.Fatalf("An error occured while trying to decode id_rsa_without_password\n")
	}

	s, err := signerFromEncryptedKey(pemBlock, []byte(""))
	if err == nil {
		t.Fatalf("signerFromEncryptedKey should return an error.\n")
	}
	if s != nil {
		t.Fatalf("signerFromEncryptedKey shouldn't return a signer.\n")
	}
}
