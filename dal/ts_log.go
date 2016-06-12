package dal

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	sqlx_types "github.com/jmoiron/sqlx/types"

	"github.com/resourced/resourced-master/querybuilder"
)

func NewTSLog(db *sqlx.DB) *TSLog {
	ts := &TSLog{}
	ts.db = db
	ts.table = "ts_logs"

	return ts
}

type AgentLogPayload struct {
	Host struct {
		Name string
		Tags map[string]string
	}
	Data struct {
		Loglines []string
		Filename string
	}
}

type TSLogRowsWithError struct {
	TSLogRows []*TSLogRow
	Error     error
}

type TSLogRow struct {
	ClusterID int64               `db:"cluster_id"`
	Created   time.Time           `db:"created"`
	Deleted   time.Time           `db:"deleted"`
	Hostname  string              `db:"hostname"`
	Tags      sqlx_types.JSONText `db:"tags"`
	Filename  string              `db:"filename"`
	Logline   string              `db:"logline"`
}

func (tsr *TSLogRow) GetTags() map[string]string {
	tags := make(map[string]string)
	tsr.Tags.Unmarshal(&tags)

	return tags
}

type TSLog struct {
	TSBase
}

func (ts *TSLog) CreateFromJSON(tx *sqlx.Tx, clusterID int64, jsonData []byte) error {
	payload := &AgentLogPayload{}

	err := json.Unmarshal(jsonData, payload)
	if err != nil {
		return err
	}

	return ts.Create(tx, clusterID, payload.Host.Name, payload.Host.Tags, payload.Data.Loglines, payload.Data.Filename)
}

// Create a new record.
func (ts *TSLog) Create(tx *sqlx.Tx, clusterID int64, hostname string, tags map[string]string, loglines []string, filename string) (err error) {
	if tx == nil {
		tx, err = ts.db.Beginx()
		if err != nil {
			logrus.Error(err)
			return err
		}
	}

	query := fmt.Sprintf("INSERT INTO %v (cluster_id, hostname, logline, filename, tags) VALUES ($1, $2, $3, $4, $5)", ts.table)

	prepared, err := ts.db.Preparex(query)
	if err != nil {
		logrus.Error(err)
		return err
	}

	for _, logline := range loglines {
		tagsInJson, err := json.Marshal(tags)
		if err != nil {
			tagsInJson = []byte("{}")
		}

		logFields := logrus.Fields{
			"Method":    "TSLog.Create",
			"Query":     query,
			"ClusterID": clusterID,
			"Hostname":  hostname,
			"Logline":   logline,
			"Filename":  filename,
			"Tags":      string(tagsInJson),
		}

		_, err = prepared.Exec(clusterID, hostname, logline, filename, tagsInJson)
		if err != nil {
			logFields["Error"] = err.Error()
			logrus.WithFields(logFields).Error("Failed to execute insert query")
			continue
		}

		logrus.WithFields(logFields).Info("Insert Query")
	}
	return tx.Commit()
}

// LastByClusterID returns the last row by cluster id.
func (ts *TSLog) LastByClusterID(tx *sqlx.Tx, clusterID int64) (*TSLogRow, error) {
	row := &TSLogRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 ORDER BY created DESC limit 1", ts.table)
	err := ts.db.Get(row, query, clusterID)

	return row, err
}

// AllByClusterIDAndRange returns all logs withing time range.
func (ts *TSLog) AllByClusterIDAndRange(tx *sqlx.Tx, clusterID int64, from, to int64) ([]*TSLogRow, error) {
	// Default is 15 minutes range
	if to == -1 {
		to = time.Now().UTC().Unix()
	}
	if from == -1 {
		from = to - 900
	}

	rows := []*TSLogRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND created >= to_timestamp($2) at time zone 'utc' AND created <= to_timestamp($3) at time zone 'utc' ORDER BY created DESC", ts.table)
	err := ts.db.Select(&rows, query, clusterID, from, to)

	if err != nil {
		err = fmt.Errorf("%v. Query: %v", err.Error(), query)
	}
	return rows, err
}

// AllByClusterIDRangeAndQuery returns all rows by cluster id, unix timestamp range, and resourced query.
func (ts *TSLog) AllByClusterIDRangeAndQuery(tx *sqlx.Tx, clusterID int64, from, to int64, resourcedQuery string) ([]*TSLogRow, error) {
	pgQuery := querybuilder.Parse(resourcedQuery)
	if pgQuery == "" {
		return ts.AllByClusterIDAndRange(tx, clusterID, from, to)
	}

	rows := []*TSLogRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND created >= to_timestamp($2) at time zone 'utc' AND created <= to_timestamp($3) at time zone 'utc' AND %v ORDER BY created DESC", ts.table, pgQuery)
	err := ts.db.Select(&rows, query, clusterID, from, to)

	if err != nil {
		err = fmt.Errorf("%v. Query: %v", err.Error(), query)
	}
	return rows, err
}

// AllByClusterIDLastRowIntervalAndQuery returns all rows by cluster id, unix timestamp range since last row, and resourced query.
func (ts *TSLog) AllByClusterIDLastRowIntervalAndQuery(tx *sqlx.Tx, clusterID int64, createdInterval, resourcedQuery string) ([]*TSLogRow, error) {
	lastRow, err := ts.LastByClusterID(tx, clusterID)
	if err != nil {
		return nil, err
	}

	pgQuery := querybuilder.Parse(resourcedQuery)
	rows := []*TSLogRow{}

	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND created >= ($2 at time zone 'utc' - INTERVAL '%v')", ts.table, createdInterval)
	if pgQuery != "" {
		query = fmt.Sprintf("%v AND %v ORDER BY created DESC", query, pgQuery)
	}

	err = ts.db.Select(&rows, query, clusterID, lastRow.Created)

	if err != nil {
		err = fmt.Errorf("%v. Query: %v", err.Error(), query)
	}
	return rows, err
}

// CountByClusterIDFromTimestampHostAndQuery returns count by cluster id, from unix timestamp, host, and resourced query.
func (ts *TSLog) CountByClusterIDFromTimestampHostAndQuery(tx *sqlx.Tx, clusterID int64, from int64, hostname, resourcedQuery string) (int64, error) {
	pgQuery := querybuilder.Parse(resourcedQuery)
	if pgQuery == "" {
		return -1, errors.New("Query is unparsable")
	}

	var count int64
	query := fmt.Sprintf("SELECT count(logline) FROM %v WHERE cluster_id=$1 AND created >= to_timestamp($2) at time zone 'utc' AND hostname=$3 AND %v", ts.table, pgQuery)
	err := ts.db.Get(&count, query, clusterID, from, hostname)

	if err != nil {
		err = fmt.Errorf("%v. Query: %v, ClusterID: %v, From: %v, Hostname: %v", err.Error(), query, clusterID, from, hostname)
	}
	return count, err
}
