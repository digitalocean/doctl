package displayers

import (
	"io"

	"github.com/digitalocean/doctl/do"
)

type DropletBackupPolicy struct {
	DropletBackupPolicies []do.DropletBackupPolicy
}

var _ Displayable = &DropletBackupPolicy{}

func (d *DropletBackupPolicy) JSON(out io.Writer) error {
	return writeJSON(d.DropletBackupPolicies, out)
}

func (d *DropletBackupPolicy) Cols() []string {
	cols := []string{
		"DropletID", "BackupEnabled", "BackupPolicyPlan", "BackupPolicyWeekday", "BackupPolicyHour",
		"BackupPolicyWindowLengthHours", "BackupPolicyRetentionPeriodDays",
		"NextBackupWindowStart", "NextBackupWindowEnd",
	}
	return cols
}

func (d *DropletBackupPolicy) ColMap() map[string]string {
	return map[string]string{
		"DropletID": "Droplet ID", "BackupEnabled": "Enabled",
		"BackupPolicyPlan": "Plan", "BackupPolicyWeekday": "Weekday", "BackupPolicyHour": "Hour",
		"BackupPolicyWindowLengthHours": "Window Length Hours", "BackupPolicyRetentionPeriodDays": "Retention Period Days",
		"NextBackupWindowStart": "Next Window Start", "NextBackupWindowEnd": "Next Window End",
	}
}

func (d *DropletBackupPolicy) KV() []map[string]any {
	out := make([]map[string]any, 0)
	for _, policy := range d.DropletBackupPolicies {
		m := map[string]any{
			"DropletID": policy.DropletID, "BackupEnabled": policy.BackupEnabled, "BackupPolicyPlan": policy.BackupPolicy.Plan,
			"BackupPolicyWeekday": policy.BackupPolicy.Weekday, "BackupPolicyHour": policy.BackupPolicy.Hour,
			"BackupPolicyWindowLengthHours": policy.BackupPolicy.WindowLengthHours, "BackupPolicyRetentionPeriodDays": policy.BackupPolicy.RetentionPeriodDays,
			"NextBackupWindowStart": policy.NextBackupWindow.Start, "NextBackupWindowEnd": policy.NextBackupWindow.End,
		}
		out = append(out, m)
	}

	return out
}
