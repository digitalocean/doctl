package displayers

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
)

type Activation struct {
	Activations []whisk.Activation
}

var _ Displayable = &Activation{}

// ColMap implements Displayable
func (a *Activation) ColMap() map[string]string {
	return map[string]string{
		"Datetime":     "Datetime",
		"Status":       "Status",
		"Kind":         "Kind",
		"Version":      "Version",
		"ActivationId": "Activation ID",
		"Start":        "Start",
		"Wait":         "Wait",
		"Duration":     "Duration",
		"Function":     "Function",
	}
}

// Cols implements Displayable
func (a *Activation) Cols() []string {
	return []string{
		"Datetime",
		"Status",
		"Kind",
		"Version",
		"ActivationId",
		"Start",
		"Wait",
		"Duration",
		"Function",
	}
}

// JSON implements Displayable
func (a *Activation) JSON(out io.Writer) error {
	return writeJSON(a, out)
}

// KV implements Displayable
func (a *Activation) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.Activations))

	for _, actv := range a.Activations {
		o := map[string]interface{}{
			"Datetime":     time.UnixMilli(actv.Start).Format("01/02 03:04:05"),
			"Status":       getActivationStatus(actv.StatusCode),
			"Kind":         getActivationAnnotationValue(actv, "kind"),
			"Version":      actv.Version,
			"ActivationId": actv.ActivationID,
			"Start":        getActivationStartType(actv),
			"Wait":         getActivationAnnotationValue(actv, "waitTime"),
			"Duration":     fmt.Sprintf("%dms", actv.Duration),
			"Function":     getActivationFunctionName(actv),
		}
		out = append(out, o)
	}
	return out
}

func getActivationStartType(a whisk.Activation) string {
	if getActivationAnnotationValue(a, "init") == "" {
		return "cold"
	}
	return "warm"
}

func getActivationFunctionName(a whisk.Activation) string {
	name := a.Name
	path := getActivationAnnotationValue(a, "path").(string)

	if path == "" {
		return name
	}

	parts := strings.Split(path, "/")

	if len(parts) == 3 {
		return parts[1] + "/" + name
	}

	return name
}

func getActivationAnnotationValue(a whisk.Activation, key string) interface{} {
	return a.Annotations.GetValue(key)
}

// converts numeric status codes to typical string
func getActivationStatus(statusCode int) string {
	switch statusCode {
	case 0:
		return "success"
	case 1:
		return "application error"
	case 2:
		return "developer error"
	case 3:
		return "system error"
	default:
		return "unknown"
	}
}
