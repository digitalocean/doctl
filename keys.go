package godo

import "fmt"

const keysBasePath = "v2/account/keys"

// KeysService handles communication with key related method of the
// DigitalOcean API.
type KeysService struct {
	client *Client
}

// Key represents a DigitalOcean Key.
type Key struct {
	ID          int    `json:"id,float64,omitempty"`
	Name        string `json:"name,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"public_key,omitempty"`
}

type keysRoot struct {
	SSHKeys []Key `json:"ssh_keys"`
}

type keyRoot struct {
	SSHKey Key `json:"ssh_key"`
}

func (s Key) String() string {
	return Stringify(s)
}

// KeyCreateRequest represents a request to create a new key.
type KeyCreateRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// List all keys
func (s *KeysService) List() ([]Key, *Response, error) {
	req, err := s.client.NewRequest("GET", keysBasePath, nil)
	if err != nil {
		return nil, nil, err
	}

	keys := new(keysRoot)
	resp, err := s.client.Do(req, keys)
	if err != nil {
		return nil, resp, err
	}

	return keys.SSHKeys, resp, err
}

// Performs a get given a path
func (s *KeysService) get(path string) (*Key, *Response, error) {
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(keyRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.SSHKey, resp, err
}

// GetByID gets a Key by id
func (s *KeysService) GetByID(keyID int) (*Key, *Response, error) {
	path := fmt.Sprintf("%s/%d", keysBasePath, keyID)
	return s.get(path)
}

// GetByFingerprint gets a Key by by fingerprint
func (s *KeysService) GetByFingerprint(fingerprint string) (*Key, *Response, error) {
	path := fmt.Sprintf("%s/%s", keysBasePath, fingerprint)
	return s.get(path)
}

// Create a key using a KeyCreateRequest
func (s *KeysService) Create(createRequest *KeyCreateRequest) (*Key, *Response, error) {
	req, err := s.client.NewRequest("POST", keysBasePath, createRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(keyRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.SSHKey, resp, err
}

// Delete key using a path
func (s *KeysService) delete(path string) (*Response, error) {
	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)

	return resp, err
}

// DeleteByID deletes a key by its id
func (s *KeysService) DeleteByID(keyID int) (*Response, error) {
	path := fmt.Sprintf("%s/%d", keysBasePath, keyID)
	return s.delete(path)
}

// DeleteByFingerprint deletes a key by its fingerprint
func (s *KeysService) DeleteByFingerprint(fingerprint string) (*Response, error) {
	path := fmt.Sprintf("%s/%s", keysBasePath, fingerprint)
	return s.delete(path)
}
