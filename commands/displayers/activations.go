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
		"ActivationID": "Activation ID",
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
		"ActivationID",
		"Start",
		"Wait",
		"Duration",
		"Function",
	}
}

// JSON implements Displayable
func (a *Activation) JSON(out io.Writer) error {
	return writeJSON(a.Activations, out)
}

// KV implements Displayable
func (a *Activation) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.Activations))

	for _, actv := range a.Activations {
		o := map[string]interface{}{
			"Datetime":     time.UnixMilli(actv.Start).Format("01/02 03:04:05"),
			"Status":       GetActivationStatus(actv.StatusCode),
			"Kind":         getActivationAnnotationValue(actv, "kind"),
			"Version":      actv.Version,
			"ActivationID": actv.ActivationID,
			"Start":        getActivationStartType(actv),
			"Wait":         getActivationAnnotationValue(actv, "waitTime"),
			"Duration":     fmt.Sprintf("%dms", actv.Duration),
			"Function":     GetActivationFunctionName(actv),
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

// Gets the full function name for the activation.
func GetActivationFunctionName(a whisk.Activation) string {
	name := a.Name
	path := getActivationAnnotationValue(a, "path")

	if path == nil {
		return name
	}

	parts := strings.Split(path.(string), "/")

	if len(parts) == 3 {
		return parts[1] + "/" + name
	}

	return name
}

func GetActivationPackageName(a whisk.Activation) string {
	name := a.Name

	if a.Annotations == nil {
		return ""
	}

	path := a.Annotations.GetValue("path")

	if path == nil {
		return name
	}

	parts := strings.Split(path.(string), "/")

	if len(parts) == 3 {
		return parts[1]
	}

	return ""
}
func getActivationAnnotationValue(a whisk.Activation, key string) interface{} {
	if a.Annotations == nil {
		return nil
	}
	return a.Annotations.GetValue(key)
}

// converts numeric status codes to typical string
func GetActivationStatus(statusCode int) string {
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
