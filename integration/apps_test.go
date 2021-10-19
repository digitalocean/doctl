package integration

import (
	"bytes"
	"encoding/json"
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
	"github.com/gorilla/websocket"
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
					Branch: "main",
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
		Phase: godo.DeploymentPhase_PendingDeploy,
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
	testDeploymentActive = &godo.Deployment{
		ID:   testDeploymentUUID,
		Spec: &testAppSpec,
		Services: []*godo.DeploymentService{{
			Name:             "service",
			SourceCommitHash: "commit",
		}},
		Cause: "Manual",
		Phase: godo.DeploymentPhase_Active,
		Progress: &godo.DeploymentProgress{
			PendingSteps: 0,
			RunningSteps: 0,
			SuccessSteps: 1,
			ErrorSteps:   0,
			TotalSteps:   1,

			Steps: []*godo.DeploymentProgressStep{{
				Name:      "name",
				Status:    "done",
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
	testAppInProgress = &godo.App{
		ID:                   testAppUUID,
		Spec:                 &testAppSpec,
		InProgressDeployment: testDeployment,
		CreatedAt:            testAppTime,
		UpdatedAt:            testAppTime,
	}
	testAppResponse = struct {
		App *godo.App `json:"app"`
	}{
		App: testApp,
	}
	testAppResponseInProgress = struct {
		App *godo.App `json:"app"`
	}{
		App: testAppInProgress,
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
	testDeploymentActiveResponse = struct {
		Deployment *godo.Deployment `json:"deployment"`
	}{
		Deployment: testDeploymentActive,
	}
	testDeploymentsResponse = struct {
		Deployments []*godo.Deployment `json:"deployments"`
	}{
		Deployments: []*godo.Deployment{testDeployment},
	}
	testRegionsResponse = struct {
		Regions []*godo.AppRegion `json:"regions"`
	}{
		Regions: []*godo.AppRegion{{
			Slug:        "ams",
			Label:       "Amsterdam",
			Flag:        "netherlands",
			Continent:   "Europe",
			DataCenters: []string{"ams3"},
			Default:     true,
		}},
	}
	testAppsOutput = `ID                                      Spec Name    Default Ingress    Active Deployment ID                    In Progress Deployment ID    Created At                       Updated At
93a37175-f520-4a12-a7ad-26e63491dbf4    test                            f4e37431-a0f4-458f-8f9f-5c9a61d8562f                                 1970-01-01 00:00:01 +0000 UTC    1970-01-01 00:00:01 +0000 UTC`
	testDeploymentsOutput = `ID                                      Cause     Progress    Created At                       Updated At
f4e37431-a0f4-458f-8f9f-5c9a61d8562f    Manual    0/1         1970-01-01 00:00:01 +0000 UTC    1970-01-01 00:00:01 +0000 UTC`
	testActiveDeploymentOutput = `ID                                      Cause     Progress    Created At                       Updated At
f4e37431-a0f4-458f-8f9f-5c9a61d8562f    Manual    1/1         1970-01-01 00:00:01 +0000 UTC    1970-01-01 00:00:01 +0000 UTC`
	testRegionsOutput = `Region    Label        Continent    Data Centers    Is Disabled?    Reason (if disabled)    Is Default?
ams       Amsterdam    Europe       [ams3]          false                                   true`
)

var _ = suite("apps/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect          *require.Assertions
		server          *httptest.Server
		deploymentCount int
		getCounter      int
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
				if getCounter > 0 {
					json.NewEncoder(w).Encode(testAppResponse)
				} else {
					json.NewEncoder(w).Encode(testAppResponseInProgress)
					getCounter++
				}
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
				if deploymentCount > 0 {
					json.NewEncoder(w).Encode(testDeploymentActiveResponse)
				} else {
					json.NewEncoder(w).Encode(testDeploymentResponse)
					deploymentCount++
				}
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
	when("the wait flag is passed", func() {
		it("creates an app and polls for status", func() {
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
				"--wait",
			)
			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expectedOutput := "Notice: App creation is in progress, waiting for app to be running\n..\nNotice: App created\n" + testAppsOutput
			expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
		})
	})
	when("the upsert flag is passed", func() {
		it("creates an app or updates if already exists", func() {
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
				"--upsert",
				"--spec",
				specFile.Name(),
			)
			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expectedOutput := "Notice: App created\n" + testAppsOutput
			expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("apps/create-upsert", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect          *require.Assertions
		server          *httptest.Server
		appsCreateCount int
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
			case "/v2/apps":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost && req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if req.Method == http.MethodPost {
					if appsCreateCount > 0 {
						var r godo.AppCreateRequest
						err := json.NewDecoder(req.Body).Decode(&r)
						if err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						assert.Equal(t, testAppSpec, *r.Spec)
						json.NewEncoder(w).Encode(testAppResponse)
					} else {
						w.WriteHeader(http.StatusConflict)
						appsCreateCount++
					}
				}
				if req.Method == http.MethodGet {
					json.NewEncoder(w).Encode(testAppsResponse)
				}
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("uses upsert to update existing app", func() {
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
			"--upsert",
			"--spec",
			specFile.Name(),
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "Notice: App already exists, updating\nNotice: App created\n" + testAppsOutput
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
		expect          *require.Assertions
		server          *httptest.Server
		getAppCounter   int
		deploymentCount int
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
				if req.Method != http.MethodPut && req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if req.Method == http.MethodPut {
					var r godo.AppUpdateRequest
					err := json.NewDecoder(req.Body).Decode(&r)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					assert.Equal(t, testAppSpec, *r.Spec)
					json.NewEncoder(w).Encode(testAppResponse)
				}
				if req.Method == http.MethodGet {
					if getAppCounter > 0 {
						json.NewEncoder(w).Encode(testAppResponse)
					} else {
						json.NewEncoder(w).Encode(testAppResponseInProgress)
						getAppCounter++
					}
				}
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
				if deploymentCount > 0 {
					json.NewEncoder(w).Encode(testDeploymentActiveResponse)
				} else {
					json.NewEncoder(w).Encode(testDeploymentResponse)
					deploymentCount++
				}
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
	when("the wait flag is passed", func() {
		it("updates an app and polls for status", func() {
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
				"--wait",
			)
			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expectedOutput := "Notice: App update is in progress, waiting for app to be running\n..\nNotice: App updated\n" + testAppsOutput
			expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
		})
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
		expect          *require.Assertions
		server          *httptest.Server
		deploymentCount int
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

				var r godo.DeploymentCreateRequest
				err := json.NewDecoder(req.Body).Decode(&r)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				json.NewEncoder(w).Encode(testDeploymentResponse)
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

				if deploymentCount > 0 {
					json.NewEncoder(w).Encode(testDeploymentActiveResponse)
				} else {
					json.NewEncoder(w).Encode(testDeploymentResponse)
					deploymentCount++
				}

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
			"--force-rebuild",
			testAppUUID,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "Notice: Deployment created\n" + testDeploymentsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})

	when("the wait flag is passed", func() {
		it("creates an app deployment and polls for status", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"apps",
				"create-deployment",
				"--wait",
				testAppUUID,
			)

			output, _ := cmd.CombinedOutput()
			//expect.NoError(err)

			expectedOutput := "Notice: App deployment is in progress, waiting for deployment to be running\n.\nNotice: Deployment created\n" + testActiveDeploymentOutput
			expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
		})
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
		expect   *require.Assertions
		server   *httptest.Server
		wsServer *httptest.Server
		upgrader = websocket.Upgrader{}
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
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
		wsServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				expect.NoError(err)
			}

			defer c.Close()
			i := 0
			finish := 5
			for {
				i++
				data := struct {
					Data string `json:"data"`
				}{
					Data: "fake logs\n",
				}
				buf := new(bytes.Buffer)
				json.NewEncoder(buf).Encode(data)

				err = c.WriteMessage(websocket.TextMessage, buf.Bytes())
				if err != nil {
					require.NoError(t, err)
				}

				if i == finish {
					break
				}
			}
		}))
		logsURL = wsServer.URL
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
			"--tail=1",
			"-f",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := "fake logs\nfake logs\nfake logs\nfake logs\nfake logs"
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/list-regions", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/regions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				json.NewEncoder(w).Encode(testRegionsResponse)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("lists regions", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps",
			"list-regions",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := testRegionsOutput
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})

var _ = suite("apps/propose", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	testAppUUID2 := "93a37175-f520-0000-0000-26e63491dbf4"

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/apps/propose":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				var r godo.AppProposeRequest
				err := json.NewDecoder(req.Body).Decode(&r)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				assert.Equal(t, &testAppSpec, r.Spec)

				switch r.AppID {
				case testAppUUID:
					json.NewEncoder(w).Encode(&godo.AppProposeResponse{
						AppIsStatic:        true,
						AppNameAvailable:   false,
						AppNameSuggestion:  "new-name",
						AppCost:            5,
						AppTierUpgradeCost: 10,
						MaxFreeStaticApps:  "3",
					})
				case testAppUUID2:
					json.NewEncoder(w).Encode(&godo.AppProposeResponse{
						AppIsStatic:          true,
						AppNameAvailable:     true,
						AppCost:              20,
						AppTierDowngradeCost: 15,
						ExistingStaticApps:   "5",
						MaxFreeStaticApps:    "3",
					})
				default:
					t.Errorf("unexpected app uuid %s", r.AppID)
				}
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("prints info about the proposed app", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "propose",
			"--spec", "-",
			"--app", testAppUUID,
		)
		byt, err := json.Marshal(testAppSpec)
		expect.NoError(err)

		cmd.Stdin = bytes.NewReader(byt)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := `App Name Available?    Suggested App Name    Is Static?    Static App Usage    $/month    $/month on higher tier    $/month on lower tier
no                     new-name              yes           0 of 3 free         5.00       10.00                     n/a`
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})

	it("prints info about the proposed app with paid static apps", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "propose",
			"--spec", "-",
			"--app", testAppUUID2,
		)
		byt, err := json.Marshal(testAppSpec)
		expect.NoError(err)

		cmd.Stdin = bytes.NewReader(byt)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expectedOutput := `App Name Available?    Is Static?    Static App Usage       $/month    $/month on higher tier    $/month on lower tier
yes                    yes           3 of 3 free, 2 paid    20.00      n/a                       15.00`
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})

	it("fails on invalid specs", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"apps", "propose",
			"--spec", "-",
			"--app", "wrong-id", // this shouldn't reach the HTTP server
		)
		testSpec := `name: test
services:
  name: service
  github:
    repo: digitalocean/doctl
`
		cmd.Stdin = strings.NewReader(testSpec)

		output, err := cmd.CombinedOutput()
		expect.Equal("exit status 1", err.Error())

		expectedOutput := "Error: parsing app spec: json: cannot unmarshal object into Go struct field AppSpec.services of type []*godo.AppServiceSpec"
		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
})
