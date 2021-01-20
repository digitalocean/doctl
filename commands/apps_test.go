package commands

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppsCommand(t *testing.T) {
	cmd := Apps()
	require.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"create",
		"get",
		"list",
		"update",
		"delete",
		"create-deployment",
		"get-deployment",
		"list-deployments",
		"list-regions",
		"logs",
		"spec",
		"tier",
	)
}

var (
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

	testAppTier = &godo.AppTier{
		Name:                 "Test",
		Slug:                 "test",
		EgressBandwidthBytes: "10240",
		BuildSeconds:         "3000",
	}

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
)

func TestRunAppsCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		specFile, err := ioutil.TempFile("", "spec")
		require.NoError(t, err)
		defer func() {
			os.Remove(specFile.Name())
			specFile.Close()
		}()

		err = json.NewEncoder(specFile).Encode(&testAppSpec)
		require.NoError(t, err)

		app := &godo.App{
			ID:        uuid.New().String(),
			Spec:      &testAppSpec,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		createReq := &godo.AppCreateRequest{
			Spec: &testAppSpec,
		}

		tm.apps.EXPECT().Create(createReq).Times(1).Return(app, nil)

		config.Doit.Set(config.NS, doctl.ArgAppSpec, specFile.Name())

		err = RunAppsCreate(config)
		require.NoError(t, err)
	})
}

func TestRunAppsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		app := &godo.App{
			ID:        uuid.New().String(),
			Spec:      &testAppSpec,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tm.apps.EXPECT().Get(app.ID).Times(1).Return(app, nil)

		config.Args = append(config.Args, app.ID)

		err := RunAppsGet(config)
		require.NoError(t, err)
	})
}

func TestRunAppsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		apps := []*godo.App{{
			ID:        uuid.New().String(),
			Spec:      &testAppSpec,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}}

		tm.apps.EXPECT().List().Times(1).Return(apps, nil)

		err := RunAppsList(config)
		require.NoError(t, err)
	})
}

func TestRunAppsUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		specFile, err := ioutil.TempFile("", "spec")
		require.NoError(t, err)
		defer func() {
			os.Remove(specFile.Name())
			specFile.Close()
		}()

		err = json.NewEncoder(specFile).Encode(&testAppSpec)
		require.NoError(t, err)

		app := &godo.App{
			ID:        uuid.New().String(),
			Spec:      &testAppSpec,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		updateReq := &godo.AppUpdateRequest{
			Spec: &testAppSpec,
		}

		tm.apps.EXPECT().Update(app.ID, updateReq).Times(1).Return(app, nil)

		config.Args = append(config.Args, app.ID)
		config.Doit.Set(config.NS, doctl.ArgAppSpec, specFile.Name())

		err = RunAppsUpdate(config)
		require.NoError(t, err)
	})
}

func TestRunAppsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		app := &godo.App{
			ID:        uuid.New().String(),
			Spec:      &testAppSpec,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tm.apps.EXPECT().Delete(app.ID).Times(1).Return(nil)

		config.Args = append(config.Args, app.ID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunAppsDelete(config)
		require.NoError(t, err)
	})
}

func TestRunAppsCreateDeployment(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		appID := uuid.New().String()
		deployment := &godo.Deployment{
			ID:   uuid.New().String(),
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
					StartedAt: time.Now(),
				}},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tm.apps.EXPECT().CreateDeployment(appID, true).Times(1).Return(deployment, nil)

		config.Args = append(config.Args, appID)
		config.Doit.Set(config.NS, doctl.ArgAppForceRebuild, true)

		err := RunAppsCreateDeployment(config)
		require.NoError(t, err)
	})
}

func TestRunAppsGetDeployment(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		appID := uuid.New().String()
		deployment := &godo.Deployment{
			ID:   uuid.New().String(),
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
					StartedAt: time.Now(),
				}},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tm.apps.EXPECT().GetDeployment(appID, deployment.ID).Times(1).Return(deployment, nil)

		config.Args = append(config.Args, appID, deployment.ID)

		err := RunAppsGetDeployment(config)
		require.NoError(t, err)
	})
}

func TestRunAppsListDeployments(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		appID := uuid.New().String()
		deployments := []*godo.Deployment{{
			ID:   uuid.New().String(),
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
					StartedAt: time.Now(),
				}},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}}

		tm.apps.EXPECT().ListDeployments(appID).Times(1).Return(deployments, nil)

		config.Args = append(config.Args, appID)

		err := RunAppsListDeployments(config)
		require.NoError(t, err)
	})
}

func TestRunAppsGetLogs(t *testing.T) {
	appID := uuid.New().String()
	deploymentID := uuid.New().String()
	component := "service"

	types := map[string]godo.AppLogType{
		"build":  godo.AppLogTypeBuild,
		"deploy": godo.AppLogTypeDeploy,
		"run":    godo.AppLogTypeRun,
	}

	for typeStr, logType := range types {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.apps.EXPECT().GetLogs(appID, deploymentID, component, logType, true).Times(1).Return(&godo.AppLogs{LiveURL: "https://proxy-apps-prod-ams3-001.ondigitalocean.app/?token=..."}, nil)

			config.Args = append(config.Args, appID, component)
			config.Doit.Set(config.NS, doctl.ArgAppDeployment, deploymentID)
			config.Doit.Set(config.NS, doctl.ArgAppLogType, typeStr)
			config.Doit.Set(config.NS, doctl.ArgAppLogFollow, true)

			err := RunAppsGetLogs(config)
			require.NoError(t, err)
		})
	}
}

const (
	validJSONSpec = `{
	"name": "test",
	"services": [
		{
			"name": "web",
			"github": {
				"repo": "digitalocean/sample-golang",
				"branch": "main"
			}
		}
	],
	"static_sites": [
		{
			"name": "static",
			"git": {
				"repo_clone_url": "git@github.com:digitalocean/sample-gatsby.git",
				"branch": "main"
			},
			"routes": [
				{
				"path": "/static"
				}
			]
		}
	]
}`
	validYAMLSpec = `
name: test
services:
- name: web
  github:
    repo: digitalocean/sample-golang
    branch: main
static_sites:
- name: static
  git:
    repo_clone_url: git@github.com:digitalocean/sample-gatsby.git
    branch: main
  routes:
  - path: /static
`
)

func Test_parseAppSpec(t *testing.T) {
	expectedSpec := &godo.AppSpec{
		Name: "test",
		Services: []*godo.AppServiceSpec{
			{
				Name: "web",
				GitHub: &godo.GitHubSourceSpec{
					Repo:   "digitalocean/sample-golang",
					Branch: "main",
				},
			},
		},
		StaticSites: []*godo.AppStaticSiteSpec{
			{
				Name: "static",
				Git: &godo.GitSourceSpec{
					RepoCloneURL: "git@github.com:digitalocean/sample-gatsby.git",
					Branch:       "main",
				},
				Routes: []*godo.AppRouteSpec{
					{Path: "/static"},
				},
			},
		},
	}

	t.Run("json", func(t *testing.T) {
		spec, err := parseAppSpec([]byte(validJSONSpec))
		require.NoError(t, err)
		assert.Equal(t, expectedSpec, spec)
	})
	t.Run("yaml", func(t *testing.T) {
		spec, err := parseAppSpec([]byte(validYAMLSpec))
		require.NoError(t, err)
		assert.Equal(t, expectedSpec, spec)
	})
	t.Run("invalid", func(t *testing.T) {
		_, err := parseAppSpec([]byte("invalid spec"))
		require.Error(t, err)
	})
}

func TestRunAppSpecValidate(t *testing.T) {
	validYAMLSpec := ``

	tcs := []struct {
		name   string
		testFn testFn
	}{
		{
			name: "stdin yaml",
			testFn: func(config *CmdConfig, tm *tcMocks) {
				config.Args = append(config.Args, "-")

				err := RunAppsSpecValidate(bytes.NewBufferString(validYAMLSpec))(config)
				require.NoError(t, err)
			},
		},
		{
			name: "stdin json",
			testFn: func(config *CmdConfig, tm *tcMocks) {
				config.Args = append(config.Args, "-")

				err := RunAppsSpecValidate(bytes.NewBufferString(validJSONSpec))(config)
				require.NoError(t, err)
			},
		},
		{
			name: "file yaml",
			testFn: func(config *CmdConfig, tm *tcMocks) {
				file, err := ioutil.TempFile("", "doctl-test")
				require.NoError(t, err)
				defer func() {
					_ = os.Remove(file.Name())
				}()
				config.Args = append(config.Args, file.Name())

				_, err = file.WriteString(validYAMLSpec)
				require.NoError(t, err)

				err = RunAppsSpecValidate(nil)(config)
				require.NoError(t, err)
			},
		},
		{
			name: "stdin invalid",
			testFn: func(config *CmdConfig, tm *tcMocks) {
				config.Args = append(config.Args, "-")

				err := RunAppsSpecValidate(bytes.NewBufferString("hello"))(config)
				require.Error(t, err)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, tc.testFn)
		})
	}
}

func TestRunAppSpecGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		app := &godo.App{
			ID:   uuid.New().String(),
			Spec: &testAppSpec,
		}

		tm.apps.EXPECT().Get(app.ID).Times(2).Return(app, nil)

		t.Run("yaml", func(t *testing.T) {
			var buf bytes.Buffer
			config.Doit.Set(config.NS, doctl.ArgFormat, "yaml")
			config.Args = append(config.Args, app.ID)
			config.Out = &buf

			err := RunAppsSpecGet(config)
			require.NoError(t, err)
			require.Equal(t, `name: test
services:
- github:
    branch: main
    repo: digitalocean/doctl
  name: service
`, buf.String())
		})

		t.Run("json", func(t *testing.T) {
			var buf bytes.Buffer
			config.Doit.Set(config.NS, doctl.ArgFormat, "json")
			config.Args = append(config.Args, app.ID)
			config.Out = &buf

			err := RunAppsSpecGet(config)
			require.NoError(t, err)
			require.Equal(t, `{
  "name": "test",
  "services": [
    {
      "name": "service",
      "github": {
        "repo": "digitalocean/doctl",
        "branch": "main"
      }
    }
  ]
}
`, buf.String())
		})
	})

	t.Run("with-deployment", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			appID := uuid.New().String()
			deployment := &godo.Deployment{
				ID:   uuid.New().String(),
				Spec: &testAppSpec,
			}

			tm.apps.EXPECT().GetDeployment(appID, deployment.ID).Times(1).Return(deployment, nil)

			var buf bytes.Buffer
			config.Doit.Set(config.NS, doctl.ArgFormat, "yaml")
			config.Doit.Set(config.NS, doctl.ArgAppDeployment, deployment.ID)
			config.Args = append(config.Args, appID)
			config.Out = &buf

			err := RunAppsSpecGet(config)
			require.NoError(t, err)
			require.Equal(t, `name: test
services:
- github:
    branch: main
    repo: digitalocean/doctl
  name: service
`, buf.String())
		})
	})
}

func TestRunAppsListRegions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		regions := []*godo.AppRegion{{
			Slug:        "ams",
			Label:       "Amsterdam",
			Flag:        "netherlands",
			Continent:   "Europe",
			DataCenters: []string{"ams3"},
			Default:     true,
		}}

		tm.apps.EXPECT().ListRegions().Times(1).Return(regions, nil)

		err := RunAppsListRegions(config)
		require.NoError(t, err)
	})
}

func TestRunAppsTierList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tiers := []*godo.AppTier{testAppTier}

		tm.apps.EXPECT().ListTiers().Times(1).Return(tiers, nil)

		err := RunAppsTierList(config)
		require.NoError(t, err)
	})
}

func TestRunAppsTierGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.apps.EXPECT().GetTier(testAppTier.Slug).Times(1).Return(testAppTier, nil)

		config.Args = append(config.Args, testAppTier.Slug)
		err := RunAppsTierGet(config)
		require.NoError(t, err)
	})
}

func TestRunAppsTierInstanceSizeList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		instanceSizes := []*godo.AppInstanceSize{testAppInstanceSize}

		tm.apps.EXPECT().ListInstanceSizes().Times(1).Return(instanceSizes, nil)

		err := RunAppsTierInstanceSizeList(config)
		require.NoError(t, err)
	})
}

func TestRunAppsTierInstanceSizeGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.apps.EXPECT().GetInstanceSize(testAppInstanceSize.Slug).Times(1).Return(testAppInstanceSize, nil)

		config.Args = append(config.Args, testAppInstanceSize.Slug)
		err := RunAppsTierInstanceSizeGet(config)
		require.NoError(t, err)
	})
}
