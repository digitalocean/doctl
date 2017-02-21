package commands

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"

	"github.com/stretchr/testify/assert"
)

var (
	testCertificate = do.Certificate{
		Certificate: &godo.Certificate{
			ID:              "892071a0-bb95-49bc-8021-3afd67a210bf",
			Name:            "web-cert-01",
			NotAfter:        "2017-02-22T00:23:00Z",
			SHA1Fingerprint: "dfcc9f57d86bf58e321c2c6c31c7a971be244ac7",
			Created:         "2017-02-08T16:02:37Z",
		},
	}

	testCertificateList = do.Certificates{testCertificate}
)

func TestCertificateCommand(t *testing.T) {
	cmd := Certificate()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "get", "create", "list", "delete")
}

func TestCertificateGetNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunCertificateGet(config)
		assert.Error(t, err)
	})
}

func TestCertificateGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cID := "892071a0-bb95-49bc-8021-3afd67a210bf"
		tm.certificates.On("Get", cID).Return(&testCertificate, nil)

		config.Args = append(config.Args, cID)

		err := RunCertificateGet(config)
		assert.NoError(t, err)
	})
}

func TestCertificatesCreate(t *testing.T) {
	privateKey := "-----BEGIN PRIVATE KEY-----"
	privateKeyPath := filepath.Join(os.TempDir(), "pkey.pem")
	pkErr := ioutil.WriteFile(privateKeyPath, []byte(privateKey), 0600)
	assert.NoError(t, pkErr)
	defer os.Remove(privateKeyPath)

	cert := "-----BEGIN CERTIFICATE-----"
	certPath := filepath.Join(os.TempDir(), "cert.crt")
	certErr := ioutil.WriteFile(certPath, []byte(cert), 0600)
	assert.NoError(t, certErr)
	defer os.Remove(certPath)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		r := godo.CertificateRequest{
			Name:             "web-cert-01",
			PrivateKey:       privateKey,
			LeafCertificate:  cert,
			CertificateChain: cert,
		}

		tm.certificates.On("Create", &r).Return(&testCertificate, nil)

		config.Doit.Set(config.NS, doctl.ArgCertificateName, "web-cert-01")
		config.Doit.Set(config.NS, doctl.ArgPrivateKeyPath, privateKeyPath)
		config.Doit.Set(config.NS, doctl.ArgLeafCertificatePath, certPath)
		config.Doit.Set(config.NS, doctl.ArgCertificateChainPath, certPath)

		err := RunCertificateCreate(config)
		assert.NoError(t, err)
	})
}

func TestCertificateList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.certificates.On("List").Return(testCertificateList, nil)

		err := RunCertificateList(config)
		assert.NoError(t, err)
	})
}

func TestCertificateDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cID := "892071a0-bb95-49bc-8021-3afd67a210bf"
		tm.certificates.On("Delete", cID).Return(nil)

		config.Args = append(config.Args, cID)
		config.Doit.Set(config.NS, doctl.ArgDeleteForce, true)

		err := RunCertificateDelete(config)
		assert.NoError(t, err)
	})
}

func TestCertificateDeleteNoID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunCertificateDelete(config)
		assert.Error(t, err)
	})
}
