package dal

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

func NewTSMetricAggr15m(db *sqlx.DB) *TSMetricAggr15m {
	ts := &TSMetricAggr15m{}
	ts.db = db
	ts.table = "ts_metrics_aggr_15m"

	return ts
}

type TSMetricAggr15mRow struct {
	ClusterID int64          `db:"cluster_id"`
	MetricID  int64          `db:"metric_id"`
	Created   time.Time      `db:"created"`
	Key       string         `db:"key"`
	Host      sql.NullString `db:"host"`
	Avg       float64        `db:"avg"`
	Max       float64        `db:"max"`
	Min       float64        `db:"min"`
	Sum       float64        `db:"sum"`
}

type TSMetricAggr15m struct {
	Base
}

func (ts *TSMetricAggr15m) metricRowsForHighchart(tx *sqlx.Tx, host string, tsMetricAggrRows []*TSMetricAggr15mRow) (*HighchartPayload, error) {
	hcPayload := &HighchartPayload{}
	hcPayload.Name = host
	hcPayload.Data = make([][]interface{}, len(tsMetricAggrRows))

	for i, tsMetricAggrRow := range tsMetricAggrRows {
		row := make([]interface{}, 2)
		row[0] = tsMetricAggrRow.Created.UnixNano() / 1000000
		row[1] = tsMetricAggrRow.Avg

		hcPayload.Data[i] = row
	}

	return hcPayload, nil
}

// InsertOrUpdate a new record.
func (ts *TSMetricAggr15m) InsertOrUpdate(tx *sqlx.Tx, clusterID, metricID int64, metricKey string, selectAggrRow *TSMetricSelectAggregateRow) (err error) {
	// Check if metricKey is correct, if not don't do anything
	if metricKey != selectAggrRow.Key {
		return nil
	}
	if selectAggrRow == nil {
		return errors.New("Aggregate row cannot be nil")
	}

	created := time.Unix(int64(selectAggrRow.CreatedUnix), 0).UTC()

	data := make(map[string]interface{})
	data["cluster_id"] = clusterID
	data["metric_id"] = metricID
	data["created"] = created
	data["key"] = selectAggrRow.Key
	data["avg"] = selectAggrRow.Avg
	data["max"] = selectAggrRow.Max
	data["min"] = selectAggrRow.Min
	data["sum"] = selectAggrRow.Sum

	if selectAggrRow.Host != "" {
		data["host"] = selectAggrRow.Host
	}

	aggrSelectRows := make([]*TSMetricAggr15mRow, 0)
	var query string

	if selectAggrRow.Host != "" {
		query = fmt.Sprintf("SELECT * from %v WHERE cluster_id=$1 AND created=$2 AND key=$3 AND host=$4 LIMIT 1", ts.table)
		err = ts.db.Select(&aggrSelectRows, query, clusterID, created, selectAggrRow.Key, selectAggrRow.Host)

	} else {
		query = fmt.Sprintf("SELECT * from %v WHERE cluster_id=$1 AND created=$2 AND key=$3 LIMIT 1", ts.table)
		err = ts.db.Select(&aggrSelectRows, query, clusterID, created, selectAggrRow.Key)
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Method":    "TSMetricAggr15m.InsertOrUpdate.Select",
			"Created":   created,
			"ClusterID": clusterID,
			"MetricID":  metricID,
			"MetricKey": metricKey,
			"Query":     query,
		}).Error(err)

		return err
	}

	if err != nil || len(aggrSelectRows) == 0 {
		_, err = ts.InsertIntoTable(tx, data)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Method":    "TSMetricAggr15m.InsertOrUpdate.Insert",
				"Created":   created,
				"ClusterID": clusterID,
				"MetricID":  metricID,
				"MetricKey": metricKey,
			}).Error(err)
		}

	} else if selectAggrRow.Host != "" {
		query := fmt.Sprintf(`UPDATE %v SET avg=$1,max=$2,min=$3,sum=$4 WHERE cluster_id=$5 AND created=$6 AND key=$7 AND host=$8`, ts.table)
		_, err = tx.Exec(query, selectAggrRow.Avg, selectAggrRow.Max, selectAggrRow.Min, selectAggrRow.Sum, clusterID, created, selectAggrRow.Key, selectAggrRow.Host)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Method":    "TSMetricAggr15m.InsertOrUpdate.Update",
				"Created":   created,
				"ClusterID": clusterID,
				"MetricID":  metricID,
				"MetricKey": metricKey,
				"Host":      selectAggrRow.Host,
				"Query":     query,
			}).Error(err)
		}

	} else {
		query := fmt.Sprintf(`UPDATE %v SET avg=$1,max=$2,min=$3,sum=$4 WHERE cluster_id=$5 AND created=$6 AND key=$7`, ts.table)
		_, err = tx.Exec(query, selectAggrRow.Avg, selectAggrRow.Max, selectAggrRow.Min, selectAggrRow.Sum, clusterID, created, selectAggrRow.Key)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Method":    "TSMetricAggr15m.InsertOrUpdate.Update",
				"Created":   created,
				"ClusterID": clusterID,
				"MetricID":  metricID,
				"MetricKey": metricKey,
				"Query":     query,
			}).Error(err)
		}
	}

	return err
}

func (ts *TSMetricAggr15m) AllByMetricIDAndInterval(tx *sqlx.Tx, clusterID, metricID int64, interval string) ([]*TSMetricAggr15mRow, error) {
	if interval == "" {
		interval = "1 hour"
	}

	rows := []*TSMetricAggr15mRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND metric_id=$2 AND created >= (NOW() at time zone 'utc' - INTERVAL '%v') AND host <> '' ORDER BY cluster_id,metric_id,created ASC", ts.table, interval)
	err := ts.db.Select(&rows, query, clusterID, metricID)

	return rows, err
}

func (ts *TSMetricAggr15m) AllByMetricIDAndIntervalForHighchart(tx *sqlx.Tx, clusterID, metricID int64, interval string) ([]*HighchartPayload, error) {
	tsMetricRows, err := ts.AllByMetricIDAndInterval(tx, clusterID, metricID, interval)
	if err != nil {
		return nil, err
	}

	// Group all TSMetricAggr15mRows per host
	mapHostsAndMetrics := make(map[string][]*TSMetricAggr15mRow)

	for _, tsMetricRow := range tsMetricRows {
		host := tsMetricRow.Host.String

		if _, ok := mapHostsAndMetrics[host]; !ok {
			mapHostsAndMetrics[host] = make([]*TSMetricAggr15mRow, 0)
		}

		mapHostsAndMetrics[host] = append(mapHostsAndMetrics[host], tsMetricRow)
	}

	// Then generate multiple Highchart payloads per all these hosts.
	highChartPayloads := make([]*HighchartPayload, 0)

	for host, tsMetricRows := range mapHostsAndMetrics {
		highChartPayload, err := ts.metricRowsForHighchart(tx, host, tsMetricRows)
		if err != nil {
			return nil, err
		}
		highChartPayloads = append(highChartPayloads, highChartPayload)
	}

	return highChartPayloads, nil
}