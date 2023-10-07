/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package displayers

import (
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

type Databases struct {
	Databases do.Databases
	Short     bool
}

var _ Displayable = &Databases{}

func (d *Databases) JSON(out io.Writer) error {
	return writeJSON(d.Databases, out)
}

func (d *Databases) Cols() []string {
	if d.Short {
		return []string{
			"ID",
			"Name",
			"Engine",
			"Version",
			"NumNodes",
			"Region",
			"Status",
			"Size",
		}
	}

	return []string{
		"ID",
		"Name",
		"Engine",
		"Version",
		"NumNodes",
		"Region",
		"Status",
		"Size",
		"URI",
		"Created",
	}
}

func (d *Databases) ColMap() map[string]string {
	if d.Short {
		return map[string]string{
			"ID":       "ID",
			"Name":     "Name",
			"Engine":   "Engine",
			"Version":  "Version",
			"NumNodes": "Number of Nodes",
			"Region":   "Region",
			"Status":   "Status",
			"Size":     "Size",
		}
	}

	return map[string]string{
		"ID":       "ID",
		"Name":     "Name",
		"Engine":   "Engine",
		"Version":  "Version",
		"NumNodes": "Number of Nodes",
		"Region":   "Region",
		"Status":   "Status",
		"Size":     "Size",
		"URI":      "URI",
		"Created":  "Created At",
	}
}

func (d *Databases) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(d.Databases))

	for _, db := range d.Databases {
		o := map[string]interface{}{
			"ID":       db.ID,
			"Name":     db.Name,
			"Engine":   db.EngineSlug,
			"Version":  db.VersionSlug,
			"NumNodes": db.NumNodes,
			"Region":   db.RegionSlug,
			"Status":   db.Status,
			"Size":     db.SizeSlug,
			"URI":      db.Connection.URI,
			"Created":  db.CreatedAt,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseBackups struct {
	DatabaseBackups do.DatabaseBackups
}

var _ Displayable = &DatabaseBackups{}

func (db *DatabaseBackups) JSON(out io.Writer) error {
	return writeJSON(db.DatabaseBackups, out)
}

func (db *DatabaseBackups) Cols() []string {
	return []string{
		"Size",
		"Created",
	}
}

func (db *DatabaseBackups) ColMap() map[string]string {
	return map[string]string{
		"Size":    "Size in Gigabytes",
		"Created": "Created At",
	}
}

func (db *DatabaseBackups) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(db.DatabaseBackups))

	for _, b := range db.DatabaseBackups {
		o := map[string]interface{}{
			"Size":    b.SizeGigabytes,
			"Created": b.CreatedAt,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseUsers struct {
	DatabaseUsers do.DatabaseUsers
}

var _ Displayable = &DatabaseUsers{}

func (du *DatabaseUsers) JSON(out io.Writer) error {
	return writeJSON(du.DatabaseUsers, out)
}

func (du *DatabaseUsers) Cols() []string {
	return []string{
		"Name",
		"Role",
		"Password",
	}
}

func (du *DatabaseUsers) ColMap() map[string]string {
	return map[string]string{
		"Name":     "Name",
		"Role":     "Role",
		"Password": "Password",
	}
}

func (du *DatabaseUsers) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(du.DatabaseUsers))

	for _, u := range du.DatabaseUsers {
		o := map[string]interface{}{
			"Role":     u.Role,
			"Name":     u.Name,
			"Password": u.Password,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseConnection struct {
	DatabaseConnection do.DatabaseConnection
}

var _ Displayable = &DatabaseConnection{}

func (dc *DatabaseConnection) JSON(out io.Writer) error {
	return writeJSON(dc.DatabaseConnection, out)
}

func (dc *DatabaseConnection) Cols() []string {
	return []string{
		"URI",
		"Database",
		"Host",
		"Port",
		"User",
		"Password",
		"SSL",
	}
}

func (dc *DatabaseConnection) ColMap() map[string]string {
	return map[string]string{
		"URI":      "URI",
		"Database": "Database",
		"Host":     "Host",
		"Port":     "Port",
		"User":     "User",
		"Password": "Password",
		"SSL":      "SSL Required",
	}
}

func (dc *DatabaseConnection) KV() []map[string]interface{} {
	c := dc.DatabaseConnection
	o := map[string]interface{}{
		"URI":      c.URI,
		"Database": c.Database,
		"Host":     c.Host,
		"Port":     c.Port,
		"User":     c.User,
		"Password": c.Password,
		"SSL":      c.SSL,
	}

	return []map[string]interface{}{o}
}

type DatabaseReplicas struct {
	DatabaseReplicas do.DatabaseReplicas
	Short            bool
}

var _ Displayable = &DatabaseReplicas{}

func (dr *DatabaseReplicas) JSON(out io.Writer) error {
	return writeJSON(dr.DatabaseReplicas, out)
}

func (dr *DatabaseReplicas) Cols() []string {
	if dr.Short {
		return []string{
			"Name",
			"ID",
			"Region",
			"Status",
		}
	}

	return []string{
		"Name",
		"ID",
		"Region",
		"Status",
		"URI",
		"Created",
	}
}

func (dr *DatabaseReplicas) ColMap() map[string]string {
	if dr.Short {
		return map[string]string{
			"Name":   "Name",
			"ID":     "ID",
			"Region": "Region",
			"Status": "Status",
		}
	}

	return map[string]string{
		"Name":    "Name",
		"ID":      "ID",
		"Region":  "Region",
		"Status":  "Status",
		"URI":     "URI",
		"Created": "Created At",
	}
}

func (dr *DatabaseReplicas) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(dr.DatabaseReplicas))

	for _, r := range dr.DatabaseReplicas {
		o := map[string]interface{}{
			"Name":    r.Name,
			"ID":      r.ID,
			"Region":  r.Region,
			"Status":  r.Status,
			"URI":     r.Connection.URI,
			"Created": r.CreatedAt,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseOptions struct {
	DatabaseOptions do.DatabaseOptions
}

var _ Displayable = &DatabaseOptions{}

func (do *DatabaseOptions) JSON(out io.Writer) error {
	return writeJSON(do.DatabaseOptions, out)
}

func (do *DatabaseOptions) Cols() []string {
	return []string{
		"Engine",
	}
}

func (do *DatabaseOptions) ColMap() map[string]string {
	return map[string]string{
		"Engine": "Engine",
	}
}

func (do *DatabaseOptions) KV() []map[string]interface{} {
	engines := make([]string, 0)
	if &do.DatabaseOptions.MongoDBOptions != nil {
		engines = append(engines, "mongodb")
	}
	if &do.DatabaseOptions.RedisOptions != nil {
		engines = append(engines, "redis")
	}
	if &do.DatabaseOptions.MySQLOptions != nil {
		engines = append(engines, "mysql")
	}
	if &do.DatabaseOptions.PostgresSQLOptions != nil {
		engines = append(engines, "pg")
	}
	if &do.DatabaseOptions.KafkaOptions != nil {
		engines = append(engines, "kafka")
	}

	out := make([]map[string]interface{}, 0, len(engines))
	for _, eng := range engines {
		o := map[string]interface{}{
			"Engine": eng,
		}
		out = append(out, o)
	}
	return out
}

type DatabaseRegionOptions struct {
	RegionMap map[string][]string
}

var _ Displayable = &DatabaseRegionOptions{}

func (dbr *DatabaseRegionOptions) JSON(out io.Writer) error {
	return writeJSON(dbr.RegionMap, out)
}

func (dbr *DatabaseRegionOptions) Cols() []string {
	return []string{
		"Region",
		"Engines",
	}
}

func (dbr *DatabaseRegionOptions) ColMap() map[string]string {
	return map[string]string{
		"Engines": "Engines",
		"Region":  "Region",
	}
}

func (dbr *DatabaseRegionOptions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	regionEngineMap := make(map[string][]string, 0)
	for eng, regions := range dbr.RegionMap {
		for _, r := range regions {
			regionEngineMap[r] = append(regionEngineMap[r], eng)
		}
	}
	for r, engines := range regionEngineMap {
		o := map[string]interface{}{
			"Region":  r,
			"Engines": "[" + strings.Join(engines, ",") + "]",
		}
		out = append(out, o)
	}
	return out
}

type DatabaseVersionOptions struct {
	VersionMap map[string][]string
}

var _ Displayable = &DatabaseVersionOptions{}

func (dbv *DatabaseVersionOptions) JSON(out io.Writer) error {
	return writeJSON(dbv.VersionMap, out)
}

func (dbv *DatabaseVersionOptions) Cols() []string {
	return []string{
		"Engine",
		"Versions",
	}
}

func (dbv *DatabaseVersionOptions) ColMap() map[string]string {
	return map[string]string{
		"Engine":   "Engine",
		"Versions": "Versions",
	}
}

func (dbv *DatabaseVersionOptions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	for eng, versions := range dbv.VersionMap {
		o := map[string]interface{}{
			"Engine":   eng,
			"Versions": "[" + strings.Join(versions, ",") + "]",
		}
		out = append(out, o)
	}
	return out
}

type DatabaseLayoutOptions struct {
	Layouts []godo.DatabaseLayout
}

var _ Displayable = &DatabaseLayoutOptions{}

func (dbl *DatabaseLayoutOptions) JSON(out io.Writer) error {
	return writeJSON(dbl.Layouts, out)
}

func (dbl *DatabaseLayoutOptions) Cols() []string {
	return []string{
		"Slug",
		"NodeNums",
	}
}

func (dbl *DatabaseLayoutOptions) ColMap() map[string]string {
	return map[string]string{
		"Slug":     "Slug",
		"NodeNums": "NodeNums",
	}
}

func (dbl *DatabaseLayoutOptions) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	slugNodeMap := make(map[string][]string, 0)
	for _, layout := range dbl.Layouts {
		for _, s := range layout.Sizes {
			slugNodeMap[s] = append(slugNodeMap[s], strconv.Itoa(layout.NodeNum))
		}
	}
	keys := make([]string, 0)
	for k, _ := range slugNodeMap {
		keys = append(keys, k)
	}
	// sort keys to have deterministic ordering
	sort.Strings(keys)

	for _, k := range keys {
		o := map[string]interface{}{
			"Slug":     k,
			"NodeNums": "[" + strings.Join(slugNodeMap[k], ",") + "]",
		}
		out = append(out, o)
	}
	return out
}

type DatabasePools struct {
	DatabasePools do.DatabasePools
}

var _ Displayable = &DatabasePools{}

func (dp *DatabasePools) JSON(out io.Writer) error {
	return writeJSON(dp.DatabasePools, out)
}

func (dp *DatabasePools) Cols() []string {
	return []string{
		"User",
		"Name",
		"Size",
		"Database",
		"Mode",
		"URI",
	}
}

func (dp *DatabasePools) ColMap() map[string]string {
	return map[string]string{
		"User":     "User",
		"Name":     "Name",
		"Size":     "Size",
		"Database": "Database",
		"Mode":     "Mode",
		"URI":      "URI",
	}
}

func (dp *DatabasePools) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(dp.DatabasePools))

	for _, p := range dp.DatabasePools {
		o := map[string]interface{}{
			"User":     p.User,
			"Name":     p.Name,
			"Size":     p.Size,
			"Database": p.Database,
			"Mode":     p.Mode,
			"URI":      p.Connection.URI,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseMaintenanceWindow struct {
	DatabaseMaintenanceWindow do.DatabaseMaintenanceWindow
}

var _ Displayable = &DatabaseMaintenanceWindow{}

func (dmw *DatabaseMaintenanceWindow) JSON(out io.Writer) error {
	return writeJSON(dmw.DatabaseMaintenanceWindow, out)
}

func (dmw *DatabaseMaintenanceWindow) Cols() []string {
	return []string{
		"Day",
		"Hour",
		"Pending",
	}
}

func (dmw *DatabaseMaintenanceWindow) ColMap() map[string]string {
	return map[string]string{
		"Day":     "Day",
		"Hour":    "Hour",
		"Pending": "Pending",
	}
}

func (dmw *DatabaseMaintenanceWindow) KV() []map[string]interface{} {
	mw := dmw.DatabaseMaintenanceWindow
	o := map[string]interface{}{
		"Day":     mw.Day,
		"Hour":    mw.Hour,
		"Pending": mw.Pending,
	}

	return []map[string]interface{}{o}
}

type DatabaseDBs struct {
	DatabaseDBs do.DatabaseDBs
}

var _ Displayable = &DatabaseDBs{}

func (db *DatabaseDBs) JSON(out io.Writer) error {
	return writeJSON(db.DatabaseDBs, out)
}

func (db *DatabaseDBs) Cols() []string {
	return []string{
		"Name",
	}
}

func (db *DatabaseDBs) ColMap() map[string]string {
	return map[string]string{
		"Name": "Name",
	}
}

func (db *DatabaseDBs) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(db.DatabaseDBs))

	for _, p := range db.DatabaseDBs {
		o := map[string]interface{}{
			"Name": p.Name,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseSQLModes struct {
	DatabaseSQLModes []string
}

var _ Displayable = &DatabaseSQLModes{}

func (dsm *DatabaseSQLModes) JSON(out io.Writer) error {
	return writeJSON(dsm.DatabaseSQLModes, out)
}

func (dsm *DatabaseSQLModes) Cols() []string {
	return []string{
		"Name",
	}
}

func (dsm *DatabaseSQLModes) ColMap() map[string]string {
	return map[string]string{
		"Name": "Name",
	}
}

func (dsm *DatabaseSQLModes) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(dsm.DatabaseSQLModes))

	for _, p := range dsm.DatabaseSQLModes {
		o := map[string]interface{}{
			"Name": p,
		}
		out = append(out, o)
	}

	return out
}

type DatabaseFirewallRules struct {
	DatabaseFirewallRules do.DatabaseFirewallRules
}

var _ Displayable = &DatabaseFirewallRules{}

func (dr *DatabaseFirewallRules) JSON(out io.Writer) error {
	return writeJSON(dr.DatabaseFirewallRules, out)
}

func (dr *DatabaseFirewallRules) Cols() []string {
	return []string{
		"UUID",
		"ClusterUUID",
		"Type",
		"Value",
	}
}

func (dr *DatabaseFirewallRules) ColMap() map[string]string {

	return map[string]string{
		"UUID":        "UUID",
		"ClusterUUID": "ClusterUUID",
		"Type":        "Type",
		"Value":       "Value",
	}
}

func (dr *DatabaseFirewallRules) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(dr.DatabaseFirewallRules))

	for _, r := range dr.DatabaseFirewallRules {
		o := map[string]interface{}{
			"UUID":        r.UUID,
			"ClusterUUID": r.ClusterUUID,
			"Type":        r.Type,
			"Value":       r.Value,
		}
		out = append(out, o)
	}

	return out
}

type MySQLConfiguration struct {
	MySQLConfiguration do.MySQLConfig
}

var _ Displayable = &MySQLConfiguration{}

func (dc *MySQLConfiguration) JSON(out io.Writer) error {
	return writeJSON(dc.MySQLConfiguration, out)
}

func (dc *MySQLConfiguration) Cols() []string {
	return []string{
		"key",
		"value",
	}
}

func (dc *MySQLConfiguration) ColMap() map[string]string {
	return map[string]string{
		"key":   "key",
		"value": "value",
	}
}

func (dc *MySQLConfiguration) KV() []map[string]interface{} {
	c := dc.MySQLConfiguration
	o := []map[string]interface{}{}
	if c.ConnectTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "ConnectTimeout",
			"value": *c.ConnectTimeout,
		})
	}
	if c.DefaultTimeZone != nil {
		o = append(o, map[string]interface{}{
			"key":   "DefaultTimeZone",
			"value": *c.DefaultTimeZone,
		})
	}
	if c.InnodbLogBufferSize != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbLogBufferSize",
			"value": *c.InnodbLogBufferSize,
		})
	}
	if c.InnodbOnlineAlterLogMaxSize != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbOnlineAlterLogMaxSize",
			"value": *c.InnodbOnlineAlterLogMaxSize,
		})
	}
	if c.InnodbLockWaitTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbLockWaitTimeout",
			"value": *c.InnodbLockWaitTimeout,
		})
	}
	if c.InteractiveTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "InteractiveTimeout",
			"value": *c.InteractiveTimeout,
		})
	}
	if c.MaxAllowedPacket != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxAllowedPacket",
			"value": *c.MaxAllowedPacket,
		})
	}
	if c.NetReadTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "NetReadTimeout",
			"value": *c.NetReadTimeout,
		})
	}
	if c.SortBufferSize != nil {
		o = append(o, map[string]interface{}{
			"key":   "SortBufferSize",
			"value": *c.SortBufferSize,
		})
	}
	if c.SQLMode != nil {
		o = append(o, map[string]interface{}{
			"key":   "SQLMode",
			"value": *c.SQLMode,
		})
	}
	if c.SQLRequirePrimaryKey != nil {
		o = append(o, map[string]interface{}{
			"key":   "SQLRequirePrimaryKey",
			"value": *c.SQLRequirePrimaryKey,
		})
	}
	if c.WaitTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "WaitTimeout",
			"value": *c.WaitTimeout,
		})
	}
	if c.NetWriteTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "NetWriteTimeout",
			"value": *c.NetWriteTimeout,
		})
	}
	if c.GroupConcatMaxLen != nil {
		o = append(o, map[string]interface{}{
			"key":   "GroupConcatMaxLen",
			"value": *c.GroupConcatMaxLen,
		})
	}
	if c.InformationSchemaStatsExpiry != nil {
		o = append(o, map[string]interface{}{
			"key":   "InformationSchemaStatsExpiry",
			"value": *c.InformationSchemaStatsExpiry,
		})
	}
	if c.InnodbFtMinTokenSize != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbFtMinTokenSize",
			"value": *c.InnodbFtMinTokenSize,
		})
	}
	if c.InnodbFtServerStopwordTable != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbFtServerStopwordTable",
			"value": *c.InnodbFtServerStopwordTable,
		})
	}
	if c.InnodbPrintAllDeadlocks != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbPrintAllDeadlocks",
			"value": *c.InnodbPrintAllDeadlocks,
		})
	}
	if c.InnodbRollbackOnTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "InnodbRollbackOnTimeout",
			"value": *c.InnodbRollbackOnTimeout,
		})
	}
	if c.InternalTmpMemStorageEngine != nil {
		o = append(o, map[string]interface{}{
			"key":   "InternalTmpMemStorageEngine",
			"value": *c.InternalTmpMemStorageEngine,
		})
	}
	if c.MaxHeapTableSize != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxHeapTableSize",
			"value": *c.MaxHeapTableSize,
		})
	}
	if c.TmpTableSize != nil {
		o = append(o, map[string]interface{}{
			"key":   "TmpTableSize",
			"value": *c.TmpTableSize,
		})
	}
	if c.SlowQueryLog != nil {
		o = append(o, map[string]interface{}{
			"key":   "SlowQueryLog",
			"value": *c.SlowQueryLog,
		})
	}
	if c.LongQueryTime != nil {
		o = append(o, map[string]interface{}{
			"key":   "LongQueryTime",
			"value": *c.LongQueryTime,
		})
	}
	if c.BackupHour != nil {
		o = append(o, map[string]interface{}{
			"key":   "BackupHour",
			"value": *c.BackupHour,
		})
	}
	if c.BackupMinute != nil {
		o = append(o, map[string]interface{}{
			"key":   "BackupMinute",
			"value": *c.BackupMinute,
		})
	}
	if c.BinlogRetentionPeriod != nil {
		o = append(o, map[string]interface{}{
			"key":   "BinlogRetentionPeriod",
			"value": *c.BinlogRetentionPeriod,
		})
	}

	return o
}

type PostgreSQLConfiguration struct {
	PostgreSQLConfig do.PostgreSQLConfig
}

var _ Displayable = &PostgreSQLConfiguration{}

func (dc *PostgreSQLConfiguration) JSON(out io.Writer) error {
	return writeJSON(dc.PostgreSQLConfig, out)
}

func (dc *PostgreSQLConfiguration) Cols() []string {
	return []string{
		"key",
		"value",
	}
}

func (dc *PostgreSQLConfiguration) ColMap() map[string]string {
	return map[string]string{
		"key":   "key",
		"value": "value",
	}
}

func (dc *PostgreSQLConfiguration) KV() []map[string]interface{} {
	c := dc.PostgreSQLConfig
	o := []map[string]interface{}{}
	if c.AutovacuumFreezeMaxAge != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumFreezeMaxAge",
			"value": *c.AutovacuumFreezeMaxAge,
		})
	}
	if c.AutovacuumMaxWorkers != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumMaxWorkers",
			"value": *c.AutovacuumMaxWorkers,
		})
	}
	if c.AutovacuumNaptime != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumNaptime",
			"value": *c.AutovacuumNaptime,
		})
	}
	if c.AutovacuumVacuumThreshold != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumVacuumThreshold",
			"value": *c.AutovacuumVacuumThreshold,
		})
	}
	if c.AutovacuumAnalyzeThreshold != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumAnalyzeThreshold",
			"value": *c.AutovacuumAnalyzeThreshold,
		})
	}
	if c.AutovacuumVacuumScaleFactor != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumVacuumScaleFactor",
			"value": *c.AutovacuumVacuumScaleFactor,
		})
	}
	if c.AutovacuumAnalyzeScaleFactor != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumAnalyzeScaleFactor",
			"value": *c.AutovacuumAnalyzeScaleFactor,
		})
	}
	if c.AutovacuumVacuumCostDelay != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumVacuumCostDelay",
			"value": *c.AutovacuumVacuumCostDelay,
		})
	}
	if c.AutovacuumVacuumCostLimit != nil {
		o = append(o, map[string]interface{}{
			"key":   "AutovacuumVacuumCostLimit",
			"value": *c.AutovacuumVacuumCostLimit,
		})
	}
	if c.BGWriterDelay != nil {
		o = append(o, map[string]interface{}{
			"key":   "BGWriterDelay",
			"value": *c.BGWriterDelay,
		})
	}
	if c.BGWriterFlushAfter != nil {
		o = append(o, map[string]interface{}{
			"key":   "BGWriterFlushAfter",
			"value": *c.BGWriterFlushAfter,
		})
	}
	if c.BGWriterLRUMaxpages != nil {
		o = append(o, map[string]interface{}{
			"key":   "BGWriterLRUMaxpages",
			"value": *c.BGWriterLRUMaxpages,
		})
	}
	if c.BGWriterLRUMultiplier != nil {
		o = append(o, map[string]interface{}{
			"key":   "BGWriterLRUMultiplier",
			"value": *c.BGWriterLRUMultiplier,
		})
	}
	if c.DeadlockTimeoutMillis != nil {
		o = append(o, map[string]interface{}{
			"key":   "DeadlockTimeoutMillis",
			"value": *c.DeadlockTimeoutMillis,
		})
	}
	if c.DefaultToastCompression != nil {
		o = append(o, map[string]interface{}{
			"key":   "DefaultToastCompression",
			"value": *c.DefaultToastCompression,
		})
	}
	if c.IdleInTransactionSessionTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "IdleInTransactionSessionTimeout",
			"value": *c.IdleInTransactionSessionTimeout,
		})
	}
	if c.JIT != nil {
		o = append(o, map[string]interface{}{
			"key":   "JIT",
			"value": *c.JIT,
		})
	}
	if c.LogAutovacuumMinDuration != nil {
		o = append(o, map[string]interface{}{
			"key":   "LogAutovacuumMinDuration",
			"value": *c.LogAutovacuumMinDuration,
		})
	}
	if c.LogErrorVerbosity != nil {
		o = append(o, map[string]interface{}{
			"key":   "LogErrorVerbosity",
			"value": *c.LogErrorVerbosity,
		})
	}
	if c.LogLinePrefix != nil {
		o = append(o, map[string]interface{}{
			"key":   "LogLinePrefix",
			"value": *c.LogLinePrefix,
		})
	}
	if c.LogMinDurationStatement != nil {
		o = append(o, map[string]interface{}{
			"key":   "LogMinDurationStatement",
			"value": *c.LogMinDurationStatement,
		})
	}
	if c.MaxFilesPerProcess != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxFilesPerProcess",
			"value": *c.MaxFilesPerProcess,
		})
	}
	if c.MaxPreparedTransactions != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxPreparedTransactions",
			"value": *c.MaxPreparedTransactions,
		})
	}
	if c.MaxPredLocksPerTransaction != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxPredLocksPerTransaction",
			"value": *c.MaxPredLocksPerTransaction,
		})
	}
	if c.MaxLocksPerTransaction != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxLocksPerTransaction",
			"value": *c.MaxLocksPerTransaction,
		})
	}
	if c.MaxStackDepth != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxStackDepth",
			"value": *c.MaxStackDepth,
		})
	}
	if c.MaxStandbyArchiveDelay != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxStandbyArchiveDelay",
			"value": *c.MaxStandbyArchiveDelay,
		})
	}
	if c.MaxStandbyStreamingDelay != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxStandbyStreamingDelay",
			"value": *c.MaxStandbyStreamingDelay,
		})
	}
	if c.MaxReplicationSlots != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxReplicationSlots",
			"value": *c.MaxReplicationSlots,
		})
	}
	if c.MaxLogicalReplicationWorkers != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxLogicalReplicationWorkers",
			"value": *c.MaxLogicalReplicationWorkers,
		})
	}
	if c.MaxParallelWorkers != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxParallelWorkers",
			"value": *c.MaxParallelWorkers,
		})
	}
	if c.MaxParallelWorkersPerGather != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxParallelWorkersPerGather",
			"value": *c.MaxParallelWorkersPerGather,
		})
	}
	if c.MaxWorkerProcesses != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxWorkerProcesses",
			"value": *c.MaxWorkerProcesses,
		})
	}
	if c.PGPartmanBGWRole != nil {
		o = append(o, map[string]interface{}{
			"key":   "PGPartmanBGWRole",
			"value": *c.PGPartmanBGWRole,
		})
	}
	if c.PGPartmanBGWInterval != nil {
		o = append(o, map[string]interface{}{
			"key":   "PGPartmanBGWInterval",
			"value": *c.PGPartmanBGWInterval,
		})
	}
	if c.PGStatStatementsTrack != nil {
		o = append(o, map[string]interface{}{
			"key":   "PGStatStatementsTrack",
			"value": *c.PGStatStatementsTrack,
		})
	}
	if c.TempFileLimit != nil {
		o = append(o, map[string]interface{}{
			"key":   "TempFileLimit",
			"value": *c.TempFileLimit,
		})
	}
	if c.Timezone != nil {
		o = append(o, map[string]interface{}{
			"key":   "Timezone",
			"value": *c.Timezone,
		})
	}
	if c.TrackActivityQuerySize != nil {
		o = append(o, map[string]interface{}{
			"key":   "TrackActivityQuerySize",
			"value": *c.TrackActivityQuerySize,
		})
	}
	if c.TrackCommitTimestamp != nil {
		o = append(o, map[string]interface{}{
			"key":   "TrackCommitTimestamp",
			"value": *c.TrackCommitTimestamp,
		})
	}
	if c.TrackFunctions != nil {
		o = append(o, map[string]interface{}{
			"key":   "TrackFunctions",
			"value": *c.TrackFunctions,
		})
	}
	if c.TrackIOTiming != nil {
		o = append(o, map[string]interface{}{
			"key":   "TrackIOTiming",
			"value": *c.TrackIOTiming,
		})
	}
	if c.MaxWalSenders != nil {
		o = append(o, map[string]interface{}{
			"key":   "MaxWalSenders",
			"value": *c.MaxWalSenders,
		})
	}
	if c.WalSenderTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "WalSenderTimeout",
			"value": *c.WalSenderTimeout,
		})
	}
	if c.WalWriterDelay != nil {
		o = append(o, map[string]interface{}{
			"key":   "WalWriterDelay",
			"value": *c.WalWriterDelay,
		})
	}
	if c.SharedBuffersPercentage != nil {
		o = append(o, map[string]interface{}{
			"key":   "SharedBuffersPercentage",
			"value": *c.SharedBuffersPercentage,
		})
	}
	if c.PgBouncer != nil {
		if c.PgBouncer.ServerResetQueryAlways != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.ServerResetQueryAlways",
				"value": *c.PgBouncer.ServerResetQueryAlways,
			})
		}
		if c.PgBouncer.IgnoreStartupParameters != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.IgnoreStartupParameters",
				"value": strings.Join(*c.PgBouncer.IgnoreStartupParameters, ","),
			})
		}
		if c.PgBouncer.MinPoolSize != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.MinPoolSize",
				"value": *c.PgBouncer.MinPoolSize,
			})
		}
		if c.PgBouncer.ServerLifetime != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.ServerLifetime",
				"value": *c.PgBouncer.ServerLifetime,
			})
		}
		if c.PgBouncer.ServerIdleTimeout != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.ServerIdleTimeout",
				"value": *c.PgBouncer.ServerIdleTimeout,
			})
		}
		if c.PgBouncer.AutodbPoolSize != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.AutodbPoolSize",
				"value": *c.PgBouncer.AutodbPoolSize,
			})
		}
		if c.PgBouncer.AutodbPoolMode != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.AutodbPoolMode",
				"value": *c.PgBouncer.AutodbPoolMode,
			})
		}
		if c.PgBouncer.AutodbMaxDbConnections != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.AutodbMaxDbConnections",
				"value": *c.PgBouncer.AutodbMaxDbConnections,
			})
		}
		if c.PgBouncer.AutodbIdleTimeout != nil {
			o = append(o, map[string]interface{}{
				"key":   "PgBouncer.AutodbIdleTimeout",
				"value": *c.PgBouncer.AutodbIdleTimeout,
			})
		}
	}
	if c.BackupHour != nil {
		o = append(o, map[string]interface{}{
			"key":   "BackupHour",
			"value": *c.BackupHour,
		})
	}
	if c.BackupMinute != nil {
		o = append(o, map[string]interface{}{
			"key":   "BackupMinute",
			"value": *c.BackupMinute,
		})
	}
	if c.WorkMem != nil {
		o = append(o, map[string]interface{}{
			"key":   "WorkMem",
			"value": *c.WorkMem,
		})
	}
	if c.TimeScaleDB != nil && c.TimeScaleDB.MaxBackgroundWorkers != nil {
		o = append(o, map[string]interface{}{
			"key":   "TimeScaleDB.MaxBackgroundWorkers",
			"value": *c.TimeScaleDB.MaxBackgroundWorkers,
		})
	}
	return o
}

type RedisConfiguration struct {
	RedisConfig do.RedisConfig
}

var _ Displayable = &RedisConfiguration{}

func (dc *RedisConfiguration) JSON(out io.Writer) error {
	return writeJSON(dc.RedisConfig, out)
}

func (dc *RedisConfiguration) Cols() []string {
	return []string{
		"key",
		"value",
	}
}

func (dc *RedisConfiguration) ColMap() map[string]string {
	return map[string]string{
		"key":   "key",
		"value": "value",
	}
}

func (dc *RedisConfiguration) KV() []map[string]interface{} {
	c := dc.RedisConfig
	o := []map[string]interface{}{}
	if c.RedisMaxmemoryPolicy != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisMaxmemoryPolicy",
			"value": *c.RedisMaxmemoryPolicy,
		})
	}
	if c.RedisPubsubClientOutputBufferLimit != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisPubsubClientOutputBufferLimit",
			"value": *c.RedisPubsubClientOutputBufferLimit,
		})
	}
	if c.RedisNumberOfDatabases != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisNumberOfDatabases",
			"value": *c.RedisNumberOfDatabases,
		})
	}
	if c.RedisIOThreads != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisIOThreads",
			"value": *c.RedisIOThreads,
		})
	}
	if c.RedisLFULogFactor != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisLFULogFactor",
			"value": *c.RedisLFULogFactor,
		})
	}
	if c.RedisLFUDecayTime != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisLFUDecayTime",
			"value": *c.RedisLFUDecayTime,
		})
	}
	if c.RedisSSL != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisSSL",
			"value": *c.RedisSSL,
		})
	}
	if c.RedisTimeout != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisTimeout",
			"value": *c.RedisTimeout,
		})
	}
	if c.RedisNotifyKeyspaceEvents != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisNotifyKeyspaceEvents",
			"value": *c.RedisNotifyKeyspaceEvents,
		})
	}
	if c.RedisPersistence != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisPersistence",
			"value": *c.RedisPersistence,
		})
	}
	if c.RedisACLChannelsDefault != nil {
		o = append(o, map[string]interface{}{
			"key":   "RedisACLChannelsDefault",
			"value": *c.RedisACLChannelsDefault,
		})
	}

	return o
}
