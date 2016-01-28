package dal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	sqlx_types "github.com/jmoiron/sqlx/types"
)

func NewWatcher(db *sqlx.DB) *Watcher {
	watcher := &Watcher{}
	watcher.db = db
	watcher.table = "watchers"
	watcher.hasID = true

	return watcher
}

type WatcherRow struct {
	ID               int64               `db:"id" json:"-"`
	ClusterID        int64               `db:"cluster_id"`
	SavedQuery       sql.NullString      `db:"saved_query"`
	Name             string              `db:"name"`
	LowAffectedHosts int64               `db:"low_affected_hosts"`
	HostsLastUpdated string              `db:"hosts_last_updated"`
	CheckInterval    string              `db:"check_interval"`
	IsSilenced       bool                `db:"is_silenced"`
	ActiveCheck      sqlx_types.JSONText `db:"active_check"`
}

func (wr *WatcherRow) IsPassive() bool {
	return wr.SavedQuery.String != ""
}

func (wr *WatcherRow) HostsLastUpdatedForPostgres() string {
	return strings.Replace(wr.HostsLastUpdated, " ago", "", -1)
}

func (wr *WatcherRow) Command() string {
	activeCheck := make(map[string]interface{})

	err := json.Unmarshal(wr.ActiveCheck, &activeCheck)
	if err != nil {
		return ""
	}

	commandInterface := activeCheck["Command"]
	if commandInterface == nil {
		return ""
	}

	return commandInterface.(string)
}

func (wr *WatcherRow) SSHUser() string {
	activeCheck := make(map[string]interface{})

	err := json.Unmarshal(wr.ActiveCheck, &activeCheck)
	if err != nil {
		return ""
	}

	sshUserInterface := activeCheck["SSHUser"]
	if sshUserInterface == nil {
		return ""
	}

	return sshUserInterface.(string)
}

type Watcher struct {
	Base
}

// All returns all watchers rows.
func (w *Watcher) All(tx *sqlx.Tx) ([]*WatcherRow, error) {
	rows := []*WatcherRow{}
	query := fmt.Sprintf("SELECT * FROM %v", w.table)
	err := w.db.Select(&rows, query)

	return rows, err
}

// AllSplitToDaemons returns all watchers rows divided into daemons equally.
func (w *Watcher) AllSplitToDaemons(tx *sqlx.Tx, daemons []string) (map[string][]*WatcherRow, error) {
	watcherRows, err := w.All(tx)
	if err != nil {
		return nil, err
	}

	buckets := make([][]*WatcherRow, len(daemons))
	for i, _ := range daemons {
		buckets[i] = make([]*WatcherRow, 0)
	}

	bucketsPointer := 0
	for _, watcherRow := range watcherRows {
		buckets[bucketsPointer] = append(buckets[bucketsPointer], watcherRow)
		bucketsPointer = bucketsPointer + 1

		if bucketsPointer >= len(buckets) {
			bucketsPointer = 0
		}
	}

	result := make(map[string][]*WatcherRow)

	for i, watchersInbucket := range buckets {
		result[daemons[i]] = watchersInbucket
	}

	return result, err
}

// AllPassiveByClusterID returns all rows by cluster_id.
func (w *Watcher) AllPassiveByClusterID(tx *sqlx.Tx, clusterID int64) ([]*WatcherRow, error) {
	rows := []*WatcherRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND saved_query <> '' ORDER BY name ASC", w.table)
	err := w.db.Select(&rows, query, clusterID)

	return rows, err
}

// AllActiveByClusterID returns all rows by cluster_id and command != nil.
func (w *Watcher) AllActiveByClusterID(tx *sqlx.Tx, clusterID int64) ([]*WatcherRow, error) {
	rows := []*WatcherRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND saved_query = '' ORDER BY name ASC", w.table)
	err := w.db.Select(&rows, query, clusterID)

	return rows, err
}

func (w *Watcher) rowFromSqlResult(tx *sqlx.Tx, sqlResult sql.Result) (*WatcherRow, error) {
	id, err := sqlResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	return w.GetByID(tx, id)
}

// GetByID returns one record by id.
func (w *Watcher) GetByID(tx *sqlx.Tx, id int64) (*WatcherRow, error) {
	watcherRow := &WatcherRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE id=$1", w.table)
	err := w.db.Get(watcherRow, query, id)

	return watcherRow, err
}

// CreateOrUpdateParameters builds params for insert or update.
func (w *Watcher) CreateOrUpdateParameters(clusterID int64, savedQuery, name string, lowAffectedHosts int64, hostsLastUpdated, checkInterval string, activeCheck map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	data["cluster_id"] = clusterID
	data["saved_query"] = savedQuery
	data["name"] = name
	data["low_affected_hosts"] = lowAffectedHosts
	data["hosts_last_updated"] = hostsLastUpdated
	data["check_interval"] = checkInterval

	activeCheckJsonBytes, err := json.Marshal(activeCheck)
	if err == nil {
		data["active_check"] = activeCheckJsonBytes
	}

	return data
}

// Create inserts one row
func (w *Watcher) Create(tx *sqlx.Tx, data map[string]interface{}) (*WatcherRow, error) {
	sqlResult, err := w.InsertIntoTable(tx, data)
	if err != nil {
		return nil, err
	}

	return w.rowFromSqlResult(tx, sqlResult)
}
