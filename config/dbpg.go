package config

import (
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// NewPGDBConfig connects to all the databases and returns them in PGDBConfig instance.
func NewPGDBConfig(generalConfig GeneralConfig) (*PGDBConfig, error) {
	conf := &PGDBConfig{}
	conf.HostByClusterID = make(map[int64]*sqlx.DB)
	conf.TSMetricByClusterID = make(map[int64]*sqlx.DB)
	conf.TSMetricAggr15mByClusterID = make(map[int64]*sqlx.DB)
	conf.TSEventByClusterID = make(map[int64]*sqlx.DB)
	conf.TSLogByClusterID = make(map[int64]*sqlx.DB)
	conf.TSCheckByClusterID = make(map[int64]*sqlx.DB)

	if strings.HasPrefix(generalConfig.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.DBMaxOpenConnections))
		}
		conf.Core = db
	}

	// ---------------------------------------------------------
	// hosts table
	//
	if strings.HasPrefix(generalConfig.Hosts.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.Hosts.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.Hosts.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.Hosts.DBMaxOpenConnections))
		}
		conf.Host = db
	}

	for clusterIDString, dsn := range generalConfig.Hosts.DSNByClusterID {
		if strings.HasPrefix(dsn, "postgres") {
			clusterID, err := strconv.ParseInt(clusterIDString, 10, 64)
			if err != nil {
				return nil, err
			}

			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if generalConfig.Hosts.DBMaxOpenConnections > int64(0) {
				db.DB.SetMaxOpenConns(int(generalConfig.Hosts.DBMaxOpenConnections))
			}
			conf.HostByClusterID[clusterID] = db
		}
	}

	// ---------------------------------------------------------
	// ts_metrics table
	//
	if strings.HasPrefix(generalConfig.Metrics.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.Metrics.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.Metrics.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.Metrics.DBMaxOpenConnections))
		}
		conf.TSMetric = db
	}

	for clusterIDString, dsn := range generalConfig.Metrics.DSNByClusterID {
		if strings.HasPrefix(dsn, "postgres") {
			clusterID, err := strconv.ParseInt(clusterIDString, 10, 64)
			if err != nil {
				return nil, err
			}

			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if generalConfig.Metrics.DBMaxOpenConnections > int64(0) {
				db.DB.SetMaxOpenConns(int(generalConfig.Metrics.DBMaxOpenConnections))
			}
			conf.TSMetricByClusterID[clusterID] = db
		}
	}

	// ---------------------------------------------------------
	// ts_metrics_aggr_15m table
	//
	if strings.HasPrefix(generalConfig.MetricsAggr15m.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.MetricsAggr15m.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.MetricsAggr15m.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.MetricsAggr15m.DBMaxOpenConnections))
		}
		conf.TSMetricAggr15m = db
	}

	for clusterIDString, dsn := range generalConfig.MetricsAggr15m.DSNByClusterID {
		if strings.HasPrefix(dsn, "postgres") {
			clusterID, err := strconv.ParseInt(clusterIDString, 10, 64)
			if err != nil {
				return nil, err
			}

			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if generalConfig.MetricsAggr15m.DBMaxOpenConnections > int64(0) {
				db.DB.SetMaxOpenConns(int(generalConfig.MetricsAggr15m.DBMaxOpenConnections))
			}
			conf.TSMetricAggr15mByClusterID[clusterID] = db
		}
	}

	// ---------------------------------------------------------
	// ts_events table
	//
	if strings.HasPrefix(generalConfig.Events.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.Events.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.Events.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.Events.DBMaxOpenConnections))
		}
		conf.TSEvent = db
	}

	for clusterIDString, dsn := range generalConfig.Events.DSNByClusterID {
		if strings.HasPrefix(dsn, "postgres") {
			clusterID, err := strconv.ParseInt(clusterIDString, 10, 64)
			if err != nil {
				return nil, err
			}

			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if generalConfig.Events.DBMaxOpenConnections > int64(0) {
				db.DB.SetMaxOpenConns(int(generalConfig.Events.DBMaxOpenConnections))
			}
			conf.TSEventByClusterID[clusterID] = db
		}
	}

	// ---------------------------------------------------------
	// ts_logs table
	//
	if strings.HasPrefix(generalConfig.Logs.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.Logs.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.Logs.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.Logs.DBMaxOpenConnections))
		}
		conf.TSLog = db
	}

	for clusterIDString, dsn := range generalConfig.Logs.DSNByClusterID {
		if strings.HasPrefix(dsn, "postgres") {
			clusterID, err := strconv.ParseInt(clusterIDString, 10, 64)
			if err != nil {
				return nil, err
			}

			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if generalConfig.Logs.DBMaxOpenConnections > int64(0) {
				db.DB.SetMaxOpenConns(int(generalConfig.Logs.DBMaxOpenConnections))
			}
			conf.TSLogByClusterID[clusterID] = db
		}
	}

	// ---------------------------------------------------------
	// ts_checks table
	//
	if strings.HasPrefix(generalConfig.Checks.DSN, "postgres") {
		db, err := sqlx.Connect("postgres", generalConfig.Checks.DSN)
		if err != nil {
			return nil, err
		}
		if generalConfig.Checks.DBMaxOpenConnections > int64(0) {
			db.DB.SetMaxOpenConns(int(generalConfig.Checks.DBMaxOpenConnections))
		}
		conf.TSCheck = db
	}

	for clusterIDString, dsn := range generalConfig.Checks.DSNByClusterID {
		if strings.HasPrefix(dsn, "postgres") {
			clusterID, err := strconv.ParseInt(clusterIDString, 10, 64)
			if err != nil {
				return nil, err
			}

			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if generalConfig.Checks.DBMaxOpenConnections > int64(0) {
				db.DB.SetMaxOpenConns(int(generalConfig.Checks.DBMaxOpenConnections))
			}
			conf.TSCheckByClusterID[clusterID] = db
		}
	}

	return conf, nil
}

// PGDBConfig stores all database configuration data.
type PGDBConfig struct {
	Core                       *sqlx.DB
	Host                       *sqlx.DB
	HostByClusterID            map[int64]*sqlx.DB
	TSMetric                   *sqlx.DB
	TSMetricByClusterID        map[int64]*sqlx.DB
	TSMetricAggr15m            *sqlx.DB
	TSMetricAggr15mByClusterID map[int64]*sqlx.DB
	TSEvent                    *sqlx.DB
	TSEventByClusterID         map[int64]*sqlx.DB
	TSLog                      *sqlx.DB
	TSLogByClusterID           map[int64]*sqlx.DB
	TSCheck                    *sqlx.DB
	TSCheckByClusterID         map[int64]*sqlx.DB
}

func (dbconf *PGDBConfig) GetHost(clusterID int64) *sqlx.DB {
	conn, ok := dbconf.HostByClusterID[clusterID]
	if !ok {
		conn = dbconf.Host
	}

	return conn
}

func (dbconf *PGDBConfig) GetTSMetric(clusterID int64) *sqlx.DB {
	conn, ok := dbconf.TSMetricByClusterID[clusterID]
	if !ok {
		conn = dbconf.TSMetric
	}

	return conn
}

func (dbconf *PGDBConfig) GetTSMetricAggr15m(clusterID int64) *sqlx.DB {
	conn, ok := dbconf.TSMetricAggr15mByClusterID[clusterID]
	if !ok {
		conn = dbconf.TSMetricAggr15m
	}

	return conn
}

func (dbconf *PGDBConfig) GetTSEvent(clusterID int64) *sqlx.DB {
	conn, ok := dbconf.TSEventByClusterID[clusterID]
	if !ok {
		conn = dbconf.TSEvent
	}

	return conn
}

func (dbconf *PGDBConfig) GetTSLog(clusterID int64) *sqlx.DB {
	conn, ok := dbconf.TSLogByClusterID[clusterID]
	if !ok {
		conn = dbconf.TSLog
	}

	return conn
}

func (dbconf *PGDBConfig) GetTSCheck(clusterID int64) *sqlx.DB {
	conn, ok := dbconf.TSCheckByClusterID[clusterID]
	if !ok {
		conn = dbconf.TSCheck
	}

	return conn
}
