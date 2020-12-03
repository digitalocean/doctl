package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var (
	testAppTier = &godo.AppTier{
		Name:                 "Test",
		Slug:                 "test",
		EgressBandwidthBytes: "10240",
		BuildSeconds:         "3000",
	}

	testAppTierResponse = struct {
		Tier *godo.AppTier `json:"tier"`
	}{testAppTier}
	testAppTiersResponse = struct {
		Tiers []*godo.AppTier `json:"tiers"`
	}{[]*godo.AppTier{testAppTier}}

	testAppTierOutput = `Name    Slug    Egress Bandwidth    Build Seconds
Test    test    10.00 KiB           3000`

	testAppInstanceSize = &godo.AppInstanceSize{
		Name:            "Basic XXS",
		Slug:            "basic-xxs",
		CPUType:         godo.AppInstanceSizeCPUType_Dedicated,
		CPUs:            "1",
		MemoryBytes:     "536870912",
		USDPerMonth:     "5",
		USDPerSecond:    "0.0000018896447",
		TierSlug:        "basic",
		TierUpgradeTo:   "professional-xs",
		TierDowngradeTo: "basic-xxxs",
	}

	testAppInstanceSizeResponse = struct {
		InstanceSize *godo.AppInstanceSize `json:"instance_size"`
	}{testAppInstanceSize}
	testAppInstanceSizesResponse = struct {
		InstanceSizes []*godo.AppInstanceSize `json:"instance_sizes"`
	}{[]*godo.AppInstanceSize{testAppInstanceSize}}

	testAppInstanceSizeOutput = `Name         Slug         CPUs           Memory        $/month    $/second     Tier     Tier Downgrade/Upgrade Path
Basic XXS    basic-xxs    1 dedicated    512.00 MiB    5          0.0000019    basic    basic-xxxs <- basic-xxs -> professional-xs`
)

var _ = suite("apps/tier/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/tiers/" + testAppTier.Slug:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppTierResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets an app tier", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"tier",
			"get",
			testAppTier.Slug,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testAppTierOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/tier/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/tiers":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppTiersResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("lists app tiers", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"tier",
			"list",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testAppTierOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/tier/instance_size/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/tiers/instance_sizes/" + testAppInstanceSize.Slug:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppInstanceSizeResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets an app instance size", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"tier",
			"instance-size",
			"get",
			testAppInstanceSize.Slug,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testAppInstanceSizeOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/tier/instance_size/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/tiers/instance_sizes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppInstanceSizesResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("lists app instance sizes", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"tier",
			"instance-size",
			"list",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testAppInstanceSizeOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})
