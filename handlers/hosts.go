package handlers

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/jmoiron/sqlx"
	"github.com/resourced/resourced-master/dal"
	"github.com/resourced/resourced-master/libhttp"
	"github.com/resourced/resourced-master/multidb"
)

func GetHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	currentUser := context.Get(r, "currentUser").(*dal.UserRow)

	currentCluster := context.Get(r, "currentCluster").(*dal.ClusterRow)

	db := context.Get(r, "db.Core").(*sqlx.DB)

	query := r.URL.Query().Get("q")

	hosts, err := dal.NewHost(db).AllByClusterIDAndQuery(nil, currentCluster.ID, query)
	if err != nil && err.Error() != "sql: no rows in result set" {
		libhttp.HandleErrorJson(w, err)
		return
	}

	savedQueries, err := dal.NewSavedQuery(db).AllByClusterID(nil, currentCluster.ID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		libhttp.HandleErrorJson(w, err)
		return
	}

	accessTokenRow, err := dal.NewAccessToken(db).GetByUserID(nil, currentUser.ID)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	metricsMap, err := dal.NewMetric(db).AllByClusterIDAsMap(nil, currentCluster.ID)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	data := struct {
		Addr               string
		CurrentUser        *dal.UserRow
		AccessToken        *dal.AccessTokenRow
		Clusters           []*dal.ClusterRow
		CurrentClusterJson string
		Hosts              []*dal.HostRow
		SavedQueries       []*dal.SavedQueryRow
		MetricsMap         map[string]int64
	}{
		context.Get(r, "addr").(string),
		currentUser,
		accessTokenRow,
		context.Get(r, "clusters").([]*dal.ClusterRow),
		string(context.Get(r, "currentClusterJson").([]byte)),
		hosts,
		savedQueries,
		metricsMap,
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/hosts/list.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, data)
}

func PostApiHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	db := context.Get(r, "db.Core").(*sqlx.DB)

	accessTokenRow := context.Get(r, "accessTokenRow").(*dal.AccessTokenRow)

	dataJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	hostRow, err := dal.NewHost(db).CreateOrUpdate(nil, accessTokenRow, dataJson)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	metricsMap, err := dal.NewMetric(db).AllByClusterIDAsMap(nil, hostRow.ClusterID)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	// Asynchronously write time series data to ts_metrics
	dbs := context.Get(r, "multidb.TSMetrics").(*multidb.MultiDB).PickMultipleForWrites()
	for _, db := range dbs {
		go func() {
			err := dal.NewTSMetric(db).CreateByHostRow(nil, hostRow, metricsMap)
			if err != nil {
				logrus.Error(err)
			}
		}()
	}

	hostRowJson, err := json.Marshal(hostRow)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	w.Write(hostRowJson)
}

func GetApiHosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	db := context.Get(r, "db.Core").(*sqlx.DB)

	accessTokenRow := context.Get(r, "accessTokenRow").(*dal.AccessTokenRow)

	query := r.URL.Query().Get("q")

	hosts, err := dal.NewHost(db).AllByClusterIDAndQuery(nil, accessTokenRow.ClusterID, query)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	hostRowsJson, err := json.Marshal(hosts)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	w.Write(hostRowsJson)
}
