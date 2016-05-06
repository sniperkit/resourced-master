package handlers

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/jmoiron/sqlx"

	"github.com/resourced/resourced-master/dal"
	"github.com/resourced/resourced-master/libhttp"
)

func GetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	qParams := r.URL.Query()

	toString := qParams.Get("To")
	if toString == "" {
		toString = qParams.Get("to")
	}
	to, err := strconv.ParseInt(toString, 10, 64)

	fromString := qParams.Get("From")
	if fromString == "" {
		fromString = qParams.Get("from")
	}
	from, err := strconv.ParseInt(fromString, 10, 64)

	query := qParams.Get("q")

	currentUser := context.Get(r, "currentUser").(*dal.UserRow)

	currentCluster := context.Get(r, "currentCluster").(*dal.ClusterRow)

	tsLogsDB := context.Get(r, "db.TSLog").(*sqlx.DB)

	var tsLogs []*dal.TSLogRow

	if fromString == "" && toString == "" {
		tsLogs, err = dal.NewTSLog(tsLogsDB).AllByClusterIDLastRowIntervalAndQuery(nil, currentCluster.ID, "15 minute", query)
		if err != nil {
			libhttp.HandleErrorJson(w, err)
			return
		}

		if len(tsLogs) > 0 {
			if from == 0 {
				from = tsLogs[0].Created.UTC().Unix()
			}
			if to == 0 {
				to = tsLogs[len(tsLogs)-1].Created.UTC().Unix()
			}
		}

	} else {
		tsLogs, err = dal.NewTSLog(tsLogsDB).AllByClusterIDRangeAndQuery(nil, currentCluster.ID, from, to, query)
		if err != nil {
			libhttp.HandleErrorJson(w, err)
			return
		}
	}

	data := struct {
		CSRFToken          string
		Addr               string
		CurrentUser        *dal.UserRow
		Clusters           []*dal.ClusterRow
		CurrentClusterJson string
		Logs               []*dal.TSLogRow
		From               int64
		To                 int64
	}{
		csrf.Token(r),
		context.Get(r, "addr").(string),
		currentUser,
		context.Get(r, "clusters").([]*dal.ClusterRow),
		string(context.Get(r, "currentClusterJson").([]byte)),
		tsLogs,
		from,
		to,
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/logs/list.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, data)
}

func GetLogsExecutors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	qParams := r.URL.Query()

	toString := qParams.Get("To")
	if toString == "" {
		toString = qParams.Get("to")
	}
	to, err := strconv.ParseInt(toString, 10, 64)
	if err != nil {
		to = time.Now().UTC().Unix()
	}

	fromString := qParams.Get("From")
	if fromString == "" {
		fromString = qParams.Get("from")
	}
	from, err := strconv.ParseInt(fromString, 10, 64)
	if err != nil {
		// default is 15 minutes range
		from = to - 900
	}

	query := qParams.Get("q")

	currentUser := context.Get(r, "currentUser").(*dal.UserRow)

	currentCluster := context.Get(r, "currentCluster").(*dal.ClusterRow)

	tsExecutorLogsDB := context.Get(r, "db.TSExecutorLog").(*sqlx.DB)

	tsLogs, err := dal.NewTSExecutorLog(tsExecutorLogsDB).AllByClusterIDRangeAndQuery(nil, currentCluster.ID, from, to, query)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	data := struct {
		CSRFToken          string
		Addr               string
		CurrentUser        *dal.UserRow
		Clusters           []*dal.ClusterRow
		CurrentClusterJson string
		Logs               []*dal.TSLogRow
		From               int64
		To                 int64
	}{
		csrf.Token(r),
		context.Get(r, "addr").(string),
		currentUser,
		context.Get(r, "clusters").([]*dal.ClusterRow),
		string(context.Get(r, "currentClusterJson").([]byte)),
		tsLogs,
		from,
		to,
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/logs/list.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, data)
}

func PostApiLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	accessTokenRow := context.Get(r, "accessTokenRow").(*dal.AccessTokenRow)

	dataJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tsLogsDB := context.Get(r, "db.TSLog").(*sqlx.DB)

	err = dal.NewTSLog(tsLogsDB).CreateFromJSON(nil, accessTokenRow.ClusterID, dataJson)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	w.Write([]byte(`{"Message": "Success"}`))
}
