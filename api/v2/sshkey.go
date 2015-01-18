package apiv2

import (
	"fmt"
	"strconv"
)

// id			number	This is a unique identification number for the key. This can be used to reference a specific SSH key when you wish to embed a key into a Droplet.
// name			string	This is the human-readable display name for the given SSH key. This is used to easily identify the SSH keys when they are displayed.
// fingerprint	string	This attribute contains the fingerprint value that is generated from the public key. This is a unique identifier that will differentiate it from other keys using a format that SSH recognizes.
// public_key	string	This attribute contains the entire public key string that was uploaded. This is what is embedded into the root user's authorized_keys file if you choose to include this SSH key during Droplet creation.
type SSHKey struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"public_key"`
	client      *Client
}

type SSHKeyResponse struct {
	SSHKey *SSHKey `json:"ssh_key"`
}

type SSHKeyList struct {
	SSHKeys []*SSHKey `json:"ssh_keys"`
	Meta    struct {
		Total int `json:"total"`
	} `json:"meta"`
}

func (c *Client) NewSSHKey() *SSHKey {
	return &SSHKey{
		client: c,
	}
}

func (c *Client) ListAllKeys() (*SSHKeyList, error) {
	var keyList SSHKeyList

	apiErr := c.Get("account/keys", nil, &keyList, nil)
	if apiErr != nil {
		return nil, fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return &keyList, nil
}

func (c *Client) GetKeyByName(name string) (*SSHKey, error) {
	keys, err := c.ListAllKeys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys.SSHKeys {
		if key.Name == name {
			return key, nil
		}
	}

	return nil, fmt.Errorf("SSH Key %s not found.", name)
}

func (c *Client) GetKeyByID(id int) (*SSHKey, error) {
	keys, err := c.ListAllKeys()
	if err != nil {
		return nil, err
	}

	for _, key := range keys.SSHKeys {
		if key.ID == id {
			return key, nil
		}
	}

	return nil, fmt.Errorf("SSH Key %s not found.", id)
}

func (c *Client) FindKey(search string) (*SSHKey, error) {
	var key *SSHKey
	var err error

	key, err = c.GetKeyByName(search)
	if err != nil {
		var searchID int64
		searchID, err = strconv.ParseInt(search, 10, 0)
		if err == nil {
			key, _ = c.GetKeyByID(int(searchID))
		}
	}

	if key == nil {
		return nil, fmt.Errorf("SSH Key %s not found.", search)
	}

	return key, nil

}

func (c *Client) CreateKey(key *SSHKey) (*SSHKey, error) {
	var keyResponse SSHKeyResponse

	apiErr := c.Post("account/keys", key, &keyResponse, nil)
	if apiErr != nil {
		return nil, fmt.Errorf("API Error: %s", apiErr.Message)
	}

	return keyResponse.SSHKey, nil
}

func (c *Client) DestroyKey(name string) error {
	keyList, err := c.ListAllKeys()
	if err != nil {
		return err
	}

	for _, key := range keyList.SSHKeys {
		if key.Name == name {
			apiErr := c.Delete(fmt.Sprintf("account/keys/%s", key.Fingerprint), nil, nil)
			if apiErr != nil {
				return fmt.Errorf("API Error: %s", apiErr.Message)
			}
			return nil
		}
	}

	return fmt.Errorf("%s Not Found.", name)
}
