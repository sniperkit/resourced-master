package cassandra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	// "strconv"
	// "strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gocql/gocql"

	"github.com/resourced/resourced-master/contexthelper"
	"github.com/resourced/resourced-master/models/cassandra/querybuilder"
)

func NewHost(ctx context.Context) *Host {
	host := &Host{}
	host.AppContext = ctx
	host.table = "hosts"

	return host
}

type AgentResourcePayload struct {
	Data map[string]string
	Host struct {
		Name string
		Tags map[string]string
	}
}

type HostRowsWithError struct {
	Hosts []*HostRow
	Error error
}

type HostRow struct {
	ID            string            `db:"id" json:"-"`
	AccessTokenID int64             `db:"access_token_id" json:"-"`
	ClusterID     int64             `db:"cluster_id"`
	Hostname      string            `db:"hostname"`
	Updated       int64             `db:"updated"`
	Tags          map[string]string `db:"tags" json:",omitempty"`
	MasterTags    map[string]string `db:"master_tags" json:",omitempty"`
}

func (h *HostRow) GetClusterID() int64 {
	return h.ClusterID
}

func (h *HostRow) GetHostname() string {
	return h.Hostname
}

func (h *HostRow) GetMasterTagsString() string {
	inJSON, err := json.Marshal(h.MasterTags)
	if err != nil {
		return ""
	}

	return string(inJSON)
}

type Host struct {
	Base
}

func (h *Host) GetCassandraSession() (*gocql.Session, error) {
	cassandradbs, err := contexthelper.GetCassandraDBConfig(h.AppContext)
	if err != nil {
		return nil, err
	}

	return cassandradbs.HostSession, nil
}

// AllCompactByClusterIDAndUpdatedInterval returns all hosts without metric data by cluster_id and updated duration.
func (h *Host) AllCompactByClusterIDAndUpdatedInterval(clusterID int64, updatedInterval string) ([]*HostRow, error) {
	session, err := h.GetCassandraSession()
	if err != nil {
		return nil, err
	}

	updatedDuration, err := time.ParseDuration(updatedInterval)
	if err != nil {
		return nil, err
	}

	updated := time.Now().UTC().Add(-1 * updatedDuration)
	updatedUnix := updated.UTC().Unix()

	rows := []*HostRow{}

	// old: 	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND updated >= (NOW() at time zone 'utc' - INTERVAL '%v')", h.table, updatedInterval)
	query := fmt.Sprintf(`SELECT id, cluster_id, access_token_id, hostname, updated, tags, master_tags FROM %v WHERE expr(idx_hosts_lucene, '{
    filter: {
        type: "boolean",
        must: [
            {type: "match", field: "cluster_id", value: %v},
            {type:"range", field:"updated", lower:%v, include_lower: true}
        ]
    }
}')`, h.table, clusterID, updatedUnix)

	var scannedClusterID, scannedAccessTokenID, scannedUpdated int64
	var scannedID, scannedHostname string
	var scannedTags, scannedMasterTags map[string]string

	iter := session.Query(query).Iter()
	for iter.Scan(&scannedID, &scannedClusterID, &scannedAccessTokenID, &scannedHostname, &scannedUpdated, &scannedTags, &scannedMasterTags) {
		rows = append(rows, &HostRow{
			ID:            scannedID,
			ClusterID:     scannedClusterID,
			AccessTokenID: scannedAccessTokenID,
			Hostname:      scannedHostname,
			Updated:       scannedUpdated,
			Tags:          scannedTags,
			MasterTags:    scannedMasterTags,
		})
	}
	if err := iter.Close(); err != nil {
		err = fmt.Errorf("%v. Query: %v", err.Error(), query)
		logrus.WithFields(logrus.Fields{"Method": "Host.AllCompactByClusterIDAndUpdatedInterval"}).Error(err)

		return nil, err
	}

	return rows, err
}

// AllCompactByClusterIDQueryAndUpdatedInterval returns all rows by resourced query.
func (h *Host) AllCompactByClusterIDQueryAndUpdatedInterval(clusterID int64, resourcedQuery, updatedInterval string) ([]*HostRow, error) {
	session, err := h.GetCassandraSession()
	if err != nil {
		return nil, err
	}

	luceneQuery := querybuilder.Parse(resourcedQuery, nil)
	if luceneQuery == "" {
		return h.AllCompactByClusterIDAndUpdatedInterval(clusterID, updatedInterval)
	}

	updatedDuration, err := time.ParseDuration(updatedInterval)
	if err != nil {
		return nil, err
	}

	updated := time.Now().UTC().Add(-1 * updatedDuration)
	updatedUnix := updated.UTC().Unix()

	rows := []*HostRow{}

	// old: 	query := fmt.Sprintf("SELECT * FROM %v WHERE cluster_id=$1 AND updated >= (NOW() at time zone 'utc' - INTERVAL '%v')", h.table, updatedInterval)
	query := fmt.Sprintf(`SELECT id, cluster_id, access_token_id, hostname, updated, tags, master_tags FROM %v WHERE expr(idx_hosts_lucene, '{
    filter: {
        type: "boolean",
        must: [
            {type: "match", field: "cluster_id", value: %v},
            {type:"range", field:"updated", lower:%v, include_lower: true},
            %v
        ]
    }
}')`, h.table, clusterID, updatedUnix, luceneQuery)

	println(query)

	var scannedClusterID, scannedAccessTokenID, scannedUpdated int64
	var scannedID, scannedHostname string
	var scannedTags, scannedMasterTags map[string]string

	iter := session.Query(query).Iter()
	for iter.Scan(&scannedID, &scannedClusterID, &scannedAccessTokenID, &scannedHostname, &scannedUpdated, &scannedTags, &scannedMasterTags) {
		rows = append(rows, &HostRow{
			ID:            scannedID,
			ClusterID:     scannedClusterID,
			AccessTokenID: scannedAccessTokenID,
			Hostname:      scannedHostname,
			Updated:       scannedUpdated,
			Tags:          scannedTags,
			MasterTags:    scannedMasterTags,
		})
	}
	if err := iter.Close(); err != nil {
		err = fmt.Errorf("%v. Query: %v", err.Error(), query)
		logrus.WithFields(logrus.Fields{"Method": "Host.AllCompactByClusterIDQueryAndUpdatedInterval"}).Error(err)

		return nil, err
	}
	return rows, err
}

// GetByID returns record by id.
func (h *Host) GetByID(id string) (*HostRow, error) {
	session, err := h.GetCassandraSession()
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT id, cluster_id, access_token_id, hostname, updated, tags, master_tags FROM %v WHERE id=?", h.table)

	var scannedClusterID, scannedAccessTokenID, scannedUpdated int64
	var scannedID, scannedHostname string
	var scannedTags, scannedMasterTags map[string]string

	err = session.Query(query, id).Scan(&scannedID, &scannedClusterID, &scannedAccessTokenID, &scannedHostname, &scannedUpdated, &scannedTags, &scannedMasterTags)
	if err != nil {
		return nil, err
	}

	row := &HostRow{
		ID:            scannedID,
		ClusterID:     scannedClusterID,
		AccessTokenID: scannedAccessTokenID,
		Hostname:      scannedHostname,
		Updated:       scannedUpdated,
		Tags:          scannedTags,
		MasterTags:    scannedMasterTags,
	}

	return row, err
}

func (h *Host) parseAgentResourcePayload(jsonData []byte) (AgentResourcePayload, error) {
	resourcedPayload := AgentResourcePayload{}

	err := json.Unmarshal(jsonData, &resourcedPayload)
	if err != nil {
		return resourcedPayload, err
	}

	return resourcedPayload, nil
}

// CreateOrUpdate performs insert/update for one host data.
func (h *Host) CreateOrUpdate(accessTokenRow *AccessTokenRow, jsonData []byte) (*HostRow, error) {
	resourcedPayload, err := h.parseAgentResourcePayload(jsonData)
	if err != nil {
		return nil, err
	}

	if resourcedPayload.Host.Name == "" {
		return nil, errors.New("Hostname cannot be empty.")
	}

	session, err := h.GetCassandraSession()
	if err != nil {
		return nil, err
	}

	id := resourcedPayload.Host.Name
	updated := time.Now().UTC().Unix()

	query := fmt.Sprintf("INSERT INTO %v (id, cluster_id, access_token_id, hostname, updated, tags) VALUES (?, ?, ?, ?, ?, ?)", h.table)

	err = session.Query(query, id, accessTokenRow.ClusterID, accessTokenRow.ID, resourcedPayload.Host.Name, updated, resourcedPayload.Host.Tags).Exec()
	if err != nil {
		return nil, err
	}

	return &HostRow{
		ID:            id,
		ClusterID:     accessTokenRow.ClusterID,
		AccessTokenID: accessTokenRow.ID,
		Hostname:      resourcedPayload.Host.Name,
		Updated:       updated,
		Tags:          resourcedPayload.Host.Tags,
	}, nil
}

// UpdateMasterTagsByID updates master tags by ID.
func (h *Host) UpdateMasterTagsByID(id string, tags map[string]string) error {
	session, err := h.GetCassandraSession()
	if err != nil {
		return err
	}

	query := fmt.Sprintf("UPDATE %v SET master_tags=? WHERE id=?", h.table)

	return session.Query(query, tags, id).Exec()
}
