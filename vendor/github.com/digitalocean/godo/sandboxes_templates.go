package godo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const sandboxTemplatesBasePath = sandboxesBasePath + "/templates"

// SandboxTemplateStatus is the state of a custom template.
type SandboxTemplateStatus string

const (
	// SandboxTemplateStatusUnspecified is the zero status.
	SandboxTemplateStatusUnspecified SandboxTemplateStatus = "TEMPLATE_STATUS_UNSPECIFIED"
	// SandboxTemplateStatusPending means the build has not started.
	SandboxTemplateStatusPending SandboxTemplateStatus = "TEMPLATE_STATUS_PENDING"
	// SandboxTemplateStatusBuilding means a build is running.
	SandboxTemplateStatusBuilding SandboxTemplateStatus = "TEMPLATE_STATUS_BUILDING"
	// SandboxTemplateStatusReady means the template can be used to create sandboxes.
	SandboxTemplateStatusReady SandboxTemplateStatus = "TEMPLATE_STATUS_READY"
	// SandboxTemplateStatusFailed means the last build failed.
	SandboxTemplateStatusFailed SandboxTemplateStatus = "TEMPLATE_STATUS_FAILED"
)

// SandboxTemplateBuildStatus is the state of one template build.
type SandboxTemplateBuildStatus string

const (
	SandboxTemplateBuildStatusUnspecified SandboxTemplateBuildStatus = "TEMPLATE_BUILD_STATUS_UNSPECIFIED"
	SandboxTemplateBuildStatusPending     SandboxTemplateBuildStatus = "TEMPLATE_BUILD_STATUS_PENDING"
	SandboxTemplateBuildStatusBuilding    SandboxTemplateBuildStatus = "TEMPLATE_BUILD_STATUS_BUILDING"
	SandboxTemplateBuildStatusSucceeded   SandboxTemplateBuildStatus = "TEMPLATE_BUILD_STATUS_SUCCEEDED"
	SandboxTemplateBuildStatusFailed      SandboxTemplateBuildStatus = "TEMPLATE_BUILD_STATUS_FAILED"
)

// SandboxImageSource identifies a customer OCI image. Auth for a private DOCR
// image is resolved server-side from the caller's team.
type SandboxImageSource struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

// SandboxTemplateSpec is what a template was built from.
type SandboxTemplateSpec struct {
	// BaseTemplate is the accepted base template the image was rebased onto.
	BaseTemplate string              `json:"base_template,omitempty"`
	Image        *SandboxImageSource `json:"image,omitempty"`
}

// SandboxTemplate is a team custom template.
type SandboxTemplate struct {
	ID        string                `json:"template_id"`
	Name      string                `json:"name"`
	Spec      *SandboxTemplateSpec  `json:"spec,omitempty"`
	Status    SandboxTemplateStatus `json:"status"`
	CreatedAt Timestamp             `json:"created_at"`
	UpdatedAt Timestamp             `json:"updated_at"`
}

// SandboxTemplateBuild is one build of a template.
type SandboxTemplateBuild struct {
	ID         string                     `json:"build_id"`
	TemplateID string                     `json:"template_id"`
	Name       string                     `json:"name,omitempty"`
	Spec       *SandboxTemplateSpec       `json:"spec,omitempty"`
	Status     SandboxTemplateBuildStatus `json:"status"`
	// Error is set when the build failed.
	Error     string    `json:"error,omitempty"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
}

// SandboxTemplateCreateRequest is the body of CreateTemplate.
type SandboxTemplateCreateRequest struct {
	// Name must not collide with a global base template name.
	Name string `json:"name"`
	// BaseTemplate is the accepted base template to rebase onto, such as
	// "coding-base".
	BaseTemplate string `json:"base_template"`
	// SourceOCIRef is the customer OCI image: registry/repository:tag or
	// registry/repository@digest.
	SourceOCIRef string `json:"source_oci_ref"`
}

// SandboxTemplateUpdateRequest is the body of UpdateTemplate. Omitted (empty)
// fields keep their previous values.
type SandboxTemplateUpdateRequest struct {
	BaseTemplate string `json:"base_template,omitempty"`
	SourceOCIRef string `json:"source_oci_ref,omitempty"`
}

// SandboxTemplateListOptions are the paging options of ListTemplates and
// ListTemplateBuilds.
type SandboxTemplateListOptions struct {
	// PageSize is the maximum number of items to return.
	PageSize int `url:"page_size,omitempty"`
	// PageToken is the NextPageToken of the previous page. Empty starts from
	// the beginning.
	PageToken string `url:"page_token,omitempty"`
}

// SandboxTemplateList is one page of templates.
type SandboxTemplateList struct {
	Templates []SandboxTemplate `json:"templates"`
	// NextPageToken is empty on the last page.
	NextPageToken string `json:"next_page_token"`
}

// SandboxTemplateBuildList is one page of template builds.
type SandboxTemplateBuildList struct {
	Builds []SandboxTemplateBuild `json:"builds"`
	// NextPageToken is empty on the last page.
	NextPageToken string `json:"next_page_token"`
}

type sandboxTemplateRoot struct {
	Template *SandboxTemplate `json:"template"`
}

// ListTemplates lists your team's custom templates. Global base templates are
// not included; pass them by name when creating a sandbox.
func (s *SandboxesServiceOp) ListTemplates(ctx context.Context, opt *SandboxTemplateListOptions) (*SandboxTemplateList, *Response, error) {
	path, err := addOptions(sandboxTemplatesBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SandboxTemplateList)
	resp, err := s.client.Do(ctx, req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// CreateTemplate creates a custom template by rebasing a customer OCI image
// onto a base template. It starts an asynchronous build: poll GetTemplate or
// ListTemplateBuilds until the template is ready.
func (s *SandboxesServiceOp) CreateTemplate(ctx context.Context, create *SandboxTemplateCreateRequest) (*SandboxTemplate, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("sandboxes: create template request is required")
	}
	return s.templateRequest(ctx, http.MethodPost, sandboxTemplatesBasePath, create)
}

// GetTemplate retrieves a custom template by ID.
func (s *SandboxesServiceOp) GetTemplate(ctx context.Context, templateID string) (*SandboxTemplate, *Response, error) {
	if templateID == "" {
		return nil, nil, errors.New("sandboxes: template id is required")
	}
	return s.templateRequest(ctx, http.MethodGet, sandboxTemplatePath(templateID), nil)
}

// UpdateTemplate changes a template's base template and/or source image and
// starts a new build.
func (s *SandboxesServiceOp) UpdateTemplate(ctx context.Context, templateID string, update *SandboxTemplateUpdateRequest) (*SandboxTemplate, *Response, error) {
	if templateID == "" {
		return nil, nil, errors.New("sandboxes: template id is required")
	}
	if update == nil {
		return nil, nil, errors.New("sandboxes: update template request is required")
	}
	return s.templateRequest(ctx, http.MethodPut, sandboxTemplatePath(templateID), update)
}

// DeleteTemplate deletes a custom template.
func (s *SandboxesServiceOp) DeleteTemplate(ctx context.Context, templateID string) (*Response, error) {
	if templateID == "" {
		return nil, errors.New("sandboxes: template id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, sandboxTemplatePath(templateID), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// ListTemplateBuilds lists the builds of a custom template.
func (s *SandboxesServiceOp) ListTemplateBuilds(ctx context.Context, templateID string, opt *SandboxTemplateListOptions) (*SandboxTemplateBuildList, *Response, error) {
	if templateID == "" {
		return nil, nil, errors.New("sandboxes: template id is required")
	}
	path, err := addOptions(sandboxTemplatePath(templateID)+"/builds", opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SandboxTemplateBuildList)
	resp, err := s.client.Do(ctx, req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// GetTemplateBuild retrieves one build of a custom template.
func (s *SandboxesServiceOp) GetTemplateBuild(ctx context.Context, templateID, buildID string) (*SandboxTemplateBuild, *Response, error) {
	if templateID == "" || buildID == "" {
		return nil, nil, errors.New("sandboxes: template id and build id are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, fmt.Sprintf("%s/builds/%s", sandboxTemplatePath(templateID), buildID), nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(struct {
		Build *SandboxTemplateBuild `json:"build"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Build == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the build")
	}
	return root.Build, resp, nil
}

func sandboxTemplatePath(templateID string) string {
	return fmt.Sprintf("%s/%s", sandboxTemplatesBasePath, templateID)
}

func (s *SandboxesServiceOp) templateRequest(ctx context.Context, method, path string, body interface{}) (*SandboxTemplate, *Response, error) {
	req, err := s.client.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(sandboxTemplateRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Template == nil {
		return nil, resp, errors.New("sandboxes: the response did not include the template")
	}
	return root.Template, resp, nil
}
