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
		"DropletID": "Droplet ID", "BackupEnabled": "Backup Enabled",
		"BackupPolicyPlan": "Backup Policy Plan", "BackupPolicyWeekday": "Backup Policy Weekday", "BackupPolicyHour": "Backup Policy Hour",
		"BackupPolicyWindowLengthHours": "Backup Policy Window Length Hours", "BackupPolicyRetentionPeriodDays": "Backup Policy Retention Period Days",
		"NextBackupWindowStart": "Next Backup Window Start", "NextBackupWindowEnd": "Next Backup Window End",
	}
}

func (d *DropletBackupPolicy) KV() []map[string]any {
	out := make([]map[string]any, 0)
	for _, d := range d.DropletBackupPolicies {
		m := map[string]any{
			"DropletID": d.DropletID, "BackupEnabled": d.BackupEnabled, "BackupPolicyPlan": d.BackupPolicy.Plan,
			"BackupPolicyWeekday": d.BackupPolicy.Weekday, "BackupPolicyHour": d.BackupPolicy.Hour,
			"BackupPolicyWindowLengthHours": d.BackupPolicy.WindowLengthHours, "BackupPolicyRetentionPeriodDays": d.BackupPolicy.RetentionPeriodDays,
			"NextBackupWindowStart": d.NextBackupWindow.Start, "NextBackupWindowEnd": d.NextBackupWindow.End,
		}
		out = append(out, m)
	}

	return out
}
