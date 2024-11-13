package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

type DropletSupportedBackupPolicy struct {
	DropletSupportedBackupPolicies []do.DropletSupportedBackupPolicy
}

var _ Displayable = &DropletSupportedBackupPolicy{}

func (d *DropletSupportedBackupPolicy) JSON(out io.Writer) error {
	return writeJSON(d.DropletSupportedBackupPolicies, out)
}

func (d *DropletSupportedBackupPolicy) Cols() []string {
	cols := []string{
		"Name", "PossibleWindowStarts", "WindowLengthHours", "RetentionPeriodDays", "PossibleDays",
	}
	return cols
}

func (d *DropletSupportedBackupPolicy) ColMap() map[string]string {
	return map[string]string{
		"Name": "Name", "PossibleWindowStarts": "Possible Window Starts",
		"WindowLengthHours": "Window Length Hours", "RetentionPeriodDays": "Retention Period Days", "PossibleDays": "Possible Days",
	}
}

func (d *DropletSupportedBackupPolicy) KV() []map[string]any {
	out := make([]map[string]any, 0)
	for _, supported := range d.DropletSupportedBackupPolicies {
		m := map[string]any{
			"Name": supported.Name, "PossibleWindowStarts": supported.PossibleWindowStarts, "WindowLengthHours": supported.WindowLengthHours,
			"RetentionPeriodDays": supported.RetentionPeriodDays, "PossibleDays": supported.PossibleDays,
		}
		out = append(out, m)
	}

	return out
}
