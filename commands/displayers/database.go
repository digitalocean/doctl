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
			"Region",
			"Status",
		}
	}

	return []string{
		"Name",
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
			"Region": "Region",
			"Status": "Status",
		}
	}

	return map[string]string{
		"Name":    "Name",
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
