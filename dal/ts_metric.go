package dal

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func NewTSMetric(db *sqlx.DB) *TSMetric {
	ts := &TSMetric{}
	ts.db = db
	ts.table = "ts_metrics"

	return ts
}

type HighchartPayload struct {
	Name string          `json:"name"`
	Data [][]interface{} `json:"data"`
}

type TSMetricRow struct {
	ClusterID int64     `db:"cluster_id"`
	MetricID  int64     `db:"metric_id"`
	Created   time.Time `db:"created"`
	Key       string    `db:"key"`
	Host      string    `db:"host"`
	Value     float64   `db:"value"`
}

type TSMetric struct {
	Base
}

func (ts *TSMetric) metricRowsForHighchart(tx *sqlx.Tx, host string, tsMetricRows []*TSMetricRow) (*HighchartPayload, error) {
	hcPayload := &HighchartPayload{}
	hcPayload.Name = host
	hcPayload.Data = make([][]interface{}, len(tsMetricRows))

	for i, tsMetricRow := range tsMetricRows {
		row := make([]interface{}, 2)
		row[0] = tsMetricRow.Created.UnixNano() / 1000000
		row[1] = tsMetricRow.Value

		hcPayload.Data[i] = row
	}

	return hcPayload, nil
}

// Create a new record.
func (ts *TSMetric) Create(tx *sqlx.Tx, clusterID, metricID int64, key, host string, value float64) error {
	insertData := make(map[string]interface{})
	insertData["cluster_id"] = clusterID
	insertData["metric_id"] = metricID
	insertData["key"] = key
	insertData["host"] = host
	insertData["value"] = value

	_, err := ts.InsertIntoTable(tx, insertData)
	return err
}

func (ts *TSMetric) CreateByHostRow(tx *sqlx.Tx, hostRow *HostRow, metricsMap map[string]int64) error {
	// Loop through every host's data and see if they are part of graph metrics.
	// If they are, insert a record in ts_metrics.
	for path, data := range hostRow.DataAsFlatKeyValue() {
		for dataKey, value := range data {
			metricKey := path + "." + dataKey

			if metricID, ok := metricsMap[metricKey]; ok {
				// Deserialized JSON number -> interface{} always have float64 as type.
				if trueValueFloat64, ok := value.(float64); ok {
					// Ignore error for now, there's no need to break the entire loop when one insert fails.
					err := ts.Create(tx, hostRow.ClusterID, metricID, metricKey, hostRow.Name, trueValueFloat64)
					if err != nil {
						println(err.Error())
					}
				}
			}
		}
	}
	return nil
}

func (ts *TSMetric) AllByMetricIDHostAndInterval(tx *sqlx.Tx, clusterID, metricID int64, host string, interval string) ([]*TSMetricRow, error) {
	if interval == "" {
		interval = "1 hour"
	}

	rows := []*TSMetricRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND metric_id=$2 AND host=$3 AND created >= (NOW() - INTERVAL '%v') ORDER BY cluster_id,metric_id,created ASC", ts.table, interval)
	err := ts.db.Select(&rows, query, clusterID, metricID, host)

	return rows, err
}

func (ts *TSMetric) AllByMetricIDHostAndIntervalForHighchart(tx *sqlx.Tx, clusterID, metricID int64, host string, interval string) (*HighchartPayload, error) {
	tsMetricRows, err := ts.AllByMetricIDHostAndInterval(tx, clusterID, metricID, host, interval)
	if err != nil {
		return nil, err
	}

	return ts.metricRowsForHighchart(tx, host, tsMetricRows)
}

func (ts *TSMetric) AllByMetricIDAndInterval(tx *sqlx.Tx, clusterID, metricID int64, interval string) ([]*TSMetricRow, error) {
	if interval == "" {
		interval = "1 hour"
	}

	rows := []*TSMetricRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND metric_id=$2 AND created >= (NOW() - INTERVAL '%v') ORDER BY cluster_id,metric_id,created ASC", ts.table, interval)
	err := ts.db.Select(&rows, query, clusterID, metricID)

	return rows, err
}

func (ts *TSMetric) AllByMetricIDAndIntervalForHighchart(tx *sqlx.Tx, clusterID, metricID int64, interval string) ([]*HighchartPayload, error) {
	tsMetricRows, err := ts.AllByMetricIDAndInterval(tx, clusterID, metricID, interval)
	if err != nil {
		return nil, err
	}

	// Group all TSMetricRows per host
	mapHostsAndMetrics := make(map[string][]*TSMetricRow)

	for _, tsMetricRow := range tsMetricRows {
		host := tsMetricRow.Host

		if _, ok := mapHostsAndMetrics[host]; !ok {
			mapHostsAndMetrics[host] = make([]*TSMetricRow, 0)
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

// DeleteByDayInterval deletes all record older than x days ago.
func (ts *TSMetric) DeleteByDayInterval(tx *sqlx.Tx, dayInterval int) error {
	if ts.table == "" {
		return errors.New("Table must not be empty.")
	}

	tx, wrapInSingleTransaction, err := ts.newTransactionIfNeeded(tx)
	if tx == nil {
		return errors.New("Transaction struct must not be empty.")
	}
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %v WHERE created < (NOW() - INTERVAL '%v day')", ts.table, dayInterval)

	_, err = tx.Exec(query)

	if wrapInSingleTransaction == true {
		err = tx.Commit()
	}

	return err
}