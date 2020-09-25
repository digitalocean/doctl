package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testAppTime = time.Unix(1, 0).UTC()
	testAppSpec = godo.AppSpec{
		Name: "test",
		Services: []*godo.AppServiceSpec{
			{
				Name: "service",
				GitHub: &godo.GitHubSourceSpec{
					Repo:   "digitalocean/doctl",
					Branch: "master",
				},
			},
		},
	}
	testDeploymentUUID = "f4e37431-a0f4-458f-8f9f-5c9a61d8562f"
	testDeployment     = &godo.Deployment{
		ID:   testDeploymentUUID,
		Spec: &testAppSpec,
		Services: []*godo.DeploymentService{{
			Name:             "service",
			SourceCommitHash: "commit",
		}},
		Cause: "Manual",
		Progress: &godo.DeploymentProgress{
			PendingSteps: 1,
			RunningSteps: 0,
			SuccessSteps: 0,
			ErrorSteps:   0,
			TotalSteps:   1,

			Steps: []*godo.DeploymentProgressStep{{
				Name:      "name",
				Status:    "pending",
				StartedAt: testAppTime,
			}},
		},
		CreatedAt: testAppTime,
		UpdatedAt: testAppTime,
	}
	testAppUUID = "93a37175-f520-4a12-a7ad-26e63491dbf4"
	testApp     = &godo.App{
		ID:               testAppUUID,
		Spec:             &testAppSpec,
		ActiveDeployment: testDeployment,
		CreatedAt:        testAppTime,
		UpdatedAt:        testAppTime,
	}
	testAppResponse = struct {
		App *godo.App `json:"app"`
	}{
		App: testApp,
	}
	testAppsResponse = struct {
		Apps []*godo.App `json:"apps"`
	}{
		Apps: []*godo.App{testApp},
	}
	testDeploymentResponse = struct {
		Deployment *godo.Deployment `json:"deployment"`
	}{
		Deployment: testDeployment,
	}
	testDeploymentsResponse = struct {
		Deployments []*godo.Deployment `json:"deployments"`
	}{
		Deployments: []*godo.Deployment{testDeployment},
	}
	testAppsOutput = `ID                                      Spec Name    Default Ingress    Active Deployment ID                    In Progress Deployment ID    Created At                       Updated At
93a37175-f520-4a12-a7ad-26e63491dbf4    test                            f4e37431-a0f4-458f-8f9f-5c9a61d8562f                                 1970-01-01 00:00:01 +0000 UTC    1970-01-01 00:00:01 +0000 UTC`
	testDeploymentsOutput = `ID                                      Cause     Progress    Created At                       Updated At
f4e37431-a0f4-458f-8f9f-5c9a61d8562f    Manual    0/1         1970-01-01 00:00:01 +0000 UTC    1970-01-01 00:00:01 +0000 UTC`
)

var _ = suite("apps/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				var r godo.AppCreateRequest
				err := json.NewDecoder(req.Body).Decode(&r)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				assert.Equal(t, testAppSpec, *r.Spec)

				json.NewEncoder(w).Encode(testAppResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("creates an app", func() {
		specFile, err := ioutil.TempFile("", "spec")
		require.NoError(t, err)
		defer func() {
			os.Remove(specFile.Name())
			specFile.Close()
		}()

		err = json.NewEncoder(specFile).Encode(&testAppSpec)
		require.NoError(t, err)

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"create",
			"--spec",
			specFile.Name(),
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "Notice: App created\n" + testAppsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets an app", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"get",
			testAppUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testAppsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppsResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("lists all app", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"list",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testAppsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				var r godo.AppUpdateRequest
				err := json.NewDecoder(req.Body).Decode(&r)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				assert.Equal(t, testAppSpec, *r.Spec)

				json.NewEncoder(w).Encode(testAppResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("updates an app", func() {
		specFile, err := ioutil.TempFile("", "spec")
		require.NoError(t, err)
		defer func() {
			os.Remove(specFile.Name())
			specFile.Close()
		}()

		err = json.NewEncoder(specFile).Encode(&testAppSpec)
		require.NoError(t, err)

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"update",
			testAppUUID,
			"--spec",
			specFile.Name(),
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "Notice: App updated\n" + testAppsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(struct {
					ID string `json:"id"`
				}{testAppUUID})
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("deletes an app", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"delete",
			testAppUUID,
			"--force",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal("Notice: App deleted", strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/create-deployment", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID + "/deployments":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testDeploymentResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("creates an app deployment", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"create-deployment",
			testAppUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "Notice: Deployment created\n" + testDeploymentsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/get-deployment", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID + "/deployments/" + testDeploymentUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testDeploymentResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets an app deployment", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"get-deployment",
			testAppUUID,
			testDeploymentUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testDeploymentsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/list-deployments", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID + "/deployments":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testDeploymentsResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("lists all app deployments", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"list-deployments",
			testAppUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testDeploymentsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/get-logs", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		var logsURL string
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/" + testAppUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testAppResponse)
			case "/v2/apps/" + testAppUUID + "/deployments/" + testDeploymentUUID + "/logs":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				assert.Equal(t, "RUN", req.URL.Query().Get("type"))
				assert.Equal(t, "true", req.URL.Query().Get("follow"))
				assert.Equal(t, "service", req.URL.Query().Get("component_name"))

				json.NewEncoder(w).Encode(&godo.AppLogs{LiveURL: logsURL})
			case "/fake-logs":
				w.Write([]byte("fake logs"))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
		logsURL = fmt.Sprintf("%s/fake-logs", server.URL)
	})

	it("gets an app's logs", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"logs",
			testAppUUID,
			"service",
			"--deployment="+testDeploymentUUID,
			"--type=run",
			"-f",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "fake logs"
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})
