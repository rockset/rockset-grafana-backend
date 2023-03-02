package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/rockset/rockset-go-client"
	api "github.com/rockset/rockset-go-client/openapi"
	opts "github.com/rockset/rockset-go-client/option"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces- only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct{}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
// TODO: Rename Datasource
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// when logging at a non-Debug level, make sure you don't include sensitive information in the message
	// (like the *backend.QueryDataRequest)
	log.DefaultLogger.Debug("QueryData called", "numQueries", len(req.Queries))
	apiKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["apiKey"]
	if !found {
		return nil, fmt.Errorf("could not locate apiKey")
	}

	server, err := getServer(req.PluginContext.DataSourceInstanceSettings.JSONData)
	if err != nil {
		return nil, fmt.Errorf("could not locate server")
	}

	rs, err := rockset.NewClient(rockset.WithAPIKey(apiKey), rockset.WithAPIServer(server))
	if err != nil {
		return nil, fmt.Errorf("could create Rockset client: %w", err)
	}

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for i, q := range req.Queries {
		log.DefaultLogger.Debug("running query", "i", i, "len", len(req.Queries))
		b, _ := json.MarshalIndent(q.JSON, "", "  ")
		log.DefaultLogger.Debug("query", "query", string(b))
		res := d.query(ctx, rs, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	IntervalMs       uint64 `json:"intervalMs"`
	MaxDataPoints    int32  `json:"maxDataPoints"`
	QueryText        string `json:"queryText"`
	QueryParamStart  string `json:"queryParamStart"`
	QueryParamStop   string `json:"queryParamStop"`
	QueryTimeField   string `json:"queryTimeField"`
	QueryLabelColumn string `json:"queryLabelColumn"`
}

func (d *Datasource) query(ctx context.Context, rs *rockset.RockClient, query backend.DataQuery) backend.DataResponse {
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel
	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %s", err.Error()))
	}

	qm.QueryParamStart = strings.TrimPrefix(qm.QueryParamStart, ":")
	qm.QueryParamStop = strings.TrimPrefix(qm.QueryParamStop, ":")

	log.DefaultLogger.Info("query",
		"interval", qm.IntervalMs,
		"max data points", qm.MaxDataPoints,
		"from", query.TimeRange.From,
		"to", query.TimeRange.To,
		"duration", query.TimeRange.To.Sub(query.TimeRange.From).String())

	log.DefaultLogger.Debug("query start", "from", query.TimeRange.From.UTC().Format(time.RFC3339))
	log.DefaultLogger.Debug("query end", "to", query.TimeRange.To.UTC().Format(time.RFC3339))
	log.DefaultLogger.Debug("query model", fmt.Sprintf("%+v", qm))
	log.DefaultLogger.Debug("query", "SQL", qm.QueryText)

	var qr api.QueryResponse
	// TODO: use a ctx to make the Query so it the query can be cancelled, but this requires updating the Go client library
	// TODO: perhaps use Grafana variables instead of query parameters?
	//   https://grafana.com/docs/grafana/latest/developers/plugins/add-support-for-variables/

	options := []opts.QueryOption{
		opts.WithParameter(qm.QueryParamStart, "timestamp", query.TimeRange.From.UTC().Format(time.RFC3339)),
		opts.WithParameter(qm.QueryParamStop, "timestamp", query.TimeRange.To.UTC().Format(time.RFC3339)),
		opts.WithRowLimit(qm.MaxDataPoints),
	}
	log.DefaultLogger.Debug("query", "SQL", qm.QueryText)
	qr, err = rs.Query(ctx, qm.QueryText, options...)
	if err != nil {
		// TODO: Should check against `AsRocksetError` equivalent?
		log.DefaultLogger.Error("query error", "error", err.Error())
		// # TODO: Catch appropriate errors
		return backend.ErrDataResponse(backend.StatusUnknown, fmt.Sprintf("query error: %s", err.Error()))
	}
	log.DefaultLogger.Info("query response", "elapsedTime", qr.Stats.ElapsedTimeMs, "results", len(qr.Results))

	labelValues, err := generateLabelValues(qm.QueryLabelColumn, qr.Results)
	if err != nil {
		errMsg := fmt.Sprintf("label generation error: %s", err.Error())
		log.DefaultLogger.Error(errMsg)
		return backend.ErrDataResponse(backend.StatusUnknown, errMsg)
	}
	log.DefaultLogger.Debug("labels", "values", labelValues)

	for label := range labelValues {
		for i, c := range qr.ColumnFields {
			// skip the time field and the label column
			if c.Name == qm.QueryTimeField || c.Name == qm.QueryLabelColumn {
				continue
			}
			log.DefaultLogger.Debug("column", "i", i, "name", c.Name, "label", label)

			// add the frames to the response
			frame, err := makeFrame(qm.QueryTimeField, c.Name, qm.QueryLabelColumn, label, qr)
			if err != nil {
				errMsg := fmt.Sprintf("failed to create frame for %s: %s", c.Name, err.Error())
				return backend.ErrDataResponse(backend.StatusUnknown, errMsg)
			}

			// only add the frame if it contains any useful data
			if frame.Fields[1].Len() > 0 {
				response.Frames = append(response.Frames, frame)
			}
		}
	}

	if len(response.Frames) == 0 {
		errMsg := fmt.Sprintf("no usable columns found in query response: %s", err.Error())
		return backend.ErrDataResponse(backend.StatusValidationFailed, errMsg)
	}

	return response
}

// extract the set of label values from the label column
func generateLabelValues(labelColumn string, results []map[string]interface{}) (map[string]bool, error) {
	labels := make(map[string]bool)

	// if there isn't any label column specified, add an empty string so we can use it as a special case
	if labelColumn == "" {
		labels[""] = true
		return labels, nil
	}

	for _, m := range results {
		label, found := m[labelColumn]
		if !found {
			log.DefaultLogger.Error("could not lookup label", "column", labelColumn)
			continue
		}
		l, ok := label.(string)
		if !ok {
			log.DefaultLogger.Error("could not cast label column value to string", "label", label)
			continue
		}
		labels[l] = true
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("could not find label column '%s' in query result", labelColumn)
	}

	return labels, nil
}

func makeFrame(timeField, valueField, labelColumn, label string, qr api.QueryResponse) (*data.Frame, error) {
	frame := data.NewFrame(valueField)

	var times []time.Time
	var values []float64
	var strValues []string

	var labels map[string]string
	// empty label means there is no label column and labels should use the zero value, which is nil
	if labelColumn != "" {
		labels = map[string]string{labelColumn: label}
	}

	// iterate over the rows
	for _, m := range qr.Results {
		if labelColumn != "" {
			if l, found := m[labelColumn]; found && l != label {
				// skip rows which doesn't match the label
				continue
			}
		}

		// the value might not be present in every row
		v, found := m[valueField]
		if !found {
			continue
		}

		// TODO: this is a bit naïve and could be improved, as the rows can be of different types
		f, ok := v.(float64)
		if ok {
			values = append(values, f)
		}
		if !ok {
			// TODO: is there a way to send warnings back to the Grafana UI?
			g, strOk := v.(string)
			if strOk {
				strValues = append(strValues, g)
			}
			if !strOk {
				log.DefaultLogger.Error("could not cast to float64 or string", "column", valueField, "value", v)
				continue
			}
		}

		// TODO: time conversion could be cached as it is parsed each call to makeFrame
		t, err := parseTime(m, timeField)
		if err != nil {
			return nil, err
		}
		times = append(times, t)
	}

	// add the time dimension
	frame.Fields = append(frame.Fields, data.NewField("time", labels, times))
	// add values
	if len(values) > 0 {
		frame.Fields = append(frame.Fields, data.NewField(valueField, labels, values))
	} else if len(strValues) > 0 {
		frame.Fields = append(frame.Fields, data.NewField(valueField, labels, strValues))
	}
	return frame, nil
}

func parseTime(fields map[string]interface{}, key string) (time.Time, error) {
	ifc, ok := fields[key]
	if !ok {
		// TODO include a list of available columns
		return time.Time{}, fmt.Errorf("could not find column %s in query response", key)
	}

	k, ok := ifc.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("could not cast %s to string", key)
	}

	t, err := time.Parse(time.RFC3339Nano, k)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to convert %s to time: %w", k, err)
	}

	return t, nil
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// when logging at a non-Debug level, make sure you don't include sensitive information in the message
	// (like the *backend.QueryDataRequest)
	log.DefaultLogger.Debug("CheckHealth called")

	response := &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Rockset datasource is working",
	}

	apiKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["apiKey"]
	if !found {
		return healthError("failed to get api key"), nil
	}

	server, err := getServer(req.PluginContext.DataSourceInstanceSettings.JSONData)
	if err != nil {
		return healthError("unable to unmarshal json"), nil
	}

	rs, err := rockset.NewClient(rockset.WithAPIKey(apiKey), rockset.WithAPIServer(server))
	if err != nil {
		return healthError("failed to create Rockset client: %s", err.Error()), nil
	}

	// validate that we can connect by getting the org info
	_, err = rs.GetOrganization(ctx)
	if err != nil {
		return healthError("failed get connect to Rockset: %s", err.Error()), nil
	}

	return response, nil
}

func getServer(data []byte) (string, error) {
	var conf struct {
		Server string `json:"server"`
	}

	if err := json.Unmarshal(data, &conf); err != nil {
		return "", err
	}
	return conf.Server, nil
}

func healthError(msg string, args ...string) *backend.CheckHealthResult {
	var message string
	if len(args) > 0 {
		message = fmt.Sprintf(msg, args)
	} else {
		message = msg
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: message,
	}
}
