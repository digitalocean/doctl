package displayers

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/digitalocean/godo"
)

type Apps []*godo.App

var _ Displayable = (*Apps)(nil)

func (a Apps) Cols() []string {
	return []string{
		"ID",
		"Spec.Name",
		"DefaultIngress",
		"ActiveDeployment.ID",
		"InProgressDeployment.ID",
		"Created",
		"Updated",
	}
}

func (a Apps) ColMap() map[string]string {
	return map[string]string{
		"ID":                      "ID",
		"Spec.Name":               "Spec Name",
		"DefaultIngress":          "Default Ingress",
		"ActiveDeployment.ID":     "Active Deployment ID",
		"InProgressDeployment.ID": "In Progress Deployment ID",
		"Created":                 "Created At",
		"Updated":                 "Updated At",
	}
}

func (a Apps) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(a))

	for i, app := range a {
		var (
			activeDeploymentID     string
			inProgressDeploymentID string
		)

		if app.ActiveDeployment != nil {
			activeDeploymentID = app.ActiveDeployment.ID
		}

		if app.InProgressDeployment != nil {
			inProgressDeploymentID = app.InProgressDeployment.ID
		}

		out[i] = map[string]interface{}{
			"ID":                      app.ID,
			"Spec.Name":               app.Spec.Name,
			"DefaultIngress":          app.DefaultIngress,
			"ActiveDeployment.ID":     activeDeploymentID,
			"InProgressDeployment.ID": inProgressDeploymentID,
			"Created":                 app.CreatedAt,
			"Updated":                 app.UpdatedAt,
		}
	}
	return out
}

func (a Apps) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(a)
}

type Deployments []*godo.Deployment

var _ Displayable = (*Deployments)(nil)

func (d Deployments) Cols() []string {
	return []string{
		"ID",
		"Cause",
		"Progress",
		"Created",
		"Updated",
	}
}

func (d Deployments) ColMap() map[string]string {
	return map[string]string{
		"ID":       "ID",
		"Cause":    "Cause",
		"Progress": "Progress",
		"Created":  "Created At",
		"Updated":  "Updated At",
	}
}

func (d Deployments) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(d))

	for i, deployment := range d {
		progress := fmt.Sprintf("%d/%d", deployment.Progress.SuccessSteps, deployment.Progress.TotalSteps)
		if deployment.Progress.ErrorSteps > 0 {
			progress = fmt.Sprintf("%s (errors: %d)", progress, deployment.Progress.ErrorSteps)
		}

		out[i] = map[string]interface{}{
			"ID":       deployment.ID,
			"Cause":    deployment.Cause,
			"Progress": progress,
			"Created":  deployment.CreatedAt,
			"Updated":  deployment.UpdatedAt,
		}
	}
	return out
}

func (d Deployments) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(d)
}

type AppRegions []*godo.AppRegion

var _ Displayable = (*AppRegions)(nil)

func (r AppRegions) Cols() []string {
	return []string{
		"Slug",
		"Label",
		"Continent",
		"DataCenters",
		"Disabled",
		"Reason",
		"Default",
	}
}

func (r AppRegions) ColMap() map[string]string {
	return map[string]string{
		"Slug":        "Region",
		"Label":       "Label",
		"Continent":   "Continent",
		"DataCenters": "Data Centers",
		"Disabled":    "Is Disabled?",
		"Reason":      "Reason (if disabled)",
		"Default":     "Is Default?",
	}
}

func (r AppRegions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, len(r))

	for i, region := range r {
		out[i] = map[string]interface{}{
			"Slug":        region.Slug,
			"Label":       region.Label,
			"Continent":   region.Continent,
			"DataCenters": region.DataCenters,
			"Disabled":    region.Disabled,
			"Reason":      region.Reason,
			"Default":     region.Default,
		}
	}
	return out
}

func (r AppRegions) JSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(r)
}
