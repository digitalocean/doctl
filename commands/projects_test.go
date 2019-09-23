package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testProject = do.Project{
		Project: &godo.Project{
			Name:        "my project",
			Description: "my project description",
			Purpose:     "my project purpose",
			Environment: "Development",
			IsDefault:   false,
		},
	}

	testProjectList = do.Projects{testProject}

	testProjectResourcesList = do.ProjectResources{
		{
			ProjectResource: &godo.ProjectResource{URN: "do:droplet:1234"},
		},
		{
			ProjectResource: &godo.ProjectResource{URN: "do:floatingip:1.2.3.4"},
		},
	}
	testProjectResourcesListSingle = do.ProjectResources{
		{
			ProjectResource: &godo.ProjectResource{URN: "do:droplet:1234"},
		},
	}
)

func TestProjectsCommand(t *testing.T) {
	cmd := Projects()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "get", "create", "update", "delete", "resources")
}

func TestProjectsList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.projects.EXPECT().List().Return(testProjectList, nil)

		err := RunProjectsList(config)
		assert.NoError(t, err)
	})
}

func TestProjectsGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tm.projects.EXPECT().Get(projectUUID).Return(&testProject, nil)

		config.Args = append(config.Args, projectUUID)

		err := RunProjectsGet(config)
		assert.NoError(t, err)
	})
}

func TestProjectsCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectCreateRequest := &godo.CreateProjectRequest{
			Name:        "project name",
			Description: "project description",
			Purpose:     "personal use",
			Environment: "Staging",
		}
		tm.projects.EXPECT().Create(projectCreateRequest).Return(&testProject, nil)

		config.Doit.Set(config.NS, doctl.ArgProjectName, "project name")
		config.Doit.Set(config.NS, doctl.ArgProjectDescription, "project description")
		config.Doit.Set(config.NS, doctl.ArgProjectPurpose, "personal use")
		config.Doit.Set(config.NS, doctl.ArgProjectEnvironment, "Staging")

		err := RunProjectsCreate(config)
		assert.NoError(t, err)
	})
}

func TestProjectsUpdateAllAttributes(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		updateReq := &godo.UpdateProjectRequest{
			Name:        "project name",
			Description: "project description",
			Purpose:     "project purpose",
			Environment: "Production",
			IsDefault:   false,
		}
		tm.projects.EXPECT().Update(projectUUID, updateReq).Return(&testProject, nil)

		config.Args = append(config.Args, projectUUID)
		config.Doit.Set(config.NS, doctl.ArgProjectName, "project name")
		config.Doit.Set(config.NS, doctl.ArgProjectDescription, "project description")
		config.Doit.Set(config.NS, doctl.ArgProjectPurpose, "project purpose")
		config.Doit.Set(config.NS, doctl.ArgProjectEnvironment, "Production")
		config.Doit.Set(config.NS, doctl.ArgProjectIsDefault, false)

		err := RunProjectsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestProjectsUpdateSomeAttributes(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		updateReq := &godo.UpdateProjectRequest{
			Name:        "project name",
			Description: "project description",
			Purpose:     nil,
			Environment: nil,
			IsDefault:   nil,
		}
		tm.projects.EXPECT().Update(projectUUID, updateReq).Return(&testProject, nil)

		config.Args = append(config.Args, projectUUID)
		config.Doit.Set(config.NS, doctl.ArgProjectName, "project name")
		config.Doit.Set(config.NS, doctl.ArgProjectDescription, "project description")

		err := RunProjectsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestProjectsUpdateOneAttribute(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		updateReq := &godo.UpdateProjectRequest{
			Name:        "project name",
			Description: nil,
			Purpose:     nil,
			Environment: nil,
			IsDefault:   nil,
		}
		tm.projects.EXPECT().Update(projectUUID, updateReq).Return(&testProject, nil)

		config.Args = append(config.Args, projectUUID)
		config.Doit.Set(config.NS, doctl.ArgProjectName, "project name")

		err := RunProjectsUpdate(config)
		assert.NoError(t, err)
	})
}

func TestProjectsDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tm.projects.EXPECT().Delete(projectUUID).Return(nil)

		config.Args = append(config.Args, projectUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunProjectsDelete(config)
		assert.NoError(t, err)
	})
}

func TestProjectResourcesCommand(t *testing.T) {
	cmd := ProjectResourcesCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "get", "assign")
}

func TestProjectResourcesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		tm.projects.EXPECT().ListResources(projectUUID).Return(testProjectResourcesList, nil)

		config.Args = append(config.Args, projectUUID)
		err := RunProjectResourcesList(config)
		assert.NoError(t, err)
	})
}

func TestProjectResourcesGetWithValidURN(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.droplets.EXPECT().Get(1234).Return(&testDroplet, nil)

		config.Args = append(config.Args, "do:droplet:1234")
		err := RunProjectResourcesGet(config)
		assert.NoError(t, err)
	})
}

func TestProjectResourcesGetWithInvalidURN(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, "fakeurn")
		err := RunProjectResourcesGet(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `URN must be in the format "do:<resource_type>:<resource_id>" but was "fakeurn"`)
	})
}

func TestProjectResourcesAssignOneResource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		urn := "do:droplet:1234"
		tm.projects.EXPECT().AssignResources(projectUUID, []string{urn}).Return(testProjectResourcesListSingle, nil)

		config.Args = append(config.Args, projectUUID)
		config.Doit.Set(config.NS, doctl.ArgProjectResource, []string{urn})

		err := RunProjectResourcesAssign(config)
		assert.NoError(t, err)
	})
}

func TestProjectResourcesAssignMultipleResources(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		projectUUID := "ab06e011-6dd1-4034-9293-201f71aba299"
		urn := "do:droplet:1234"
		otherURN := "do:floatingip:1.2.3.4"
		tm.projects.EXPECT().AssignResources(projectUUID, []string{urn, otherURN}).Return(testProjectResourcesList, nil)

		config.Args = append(config.Args, projectUUID)
		config.Doit.Set(config.NS, doctl.ArgProjectResource, []string{urn, otherURN})

		err := RunProjectResourcesAssign(config)
		assert.NoError(t, err)
	})
}
