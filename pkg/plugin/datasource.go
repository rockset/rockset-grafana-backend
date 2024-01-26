package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/rockset/rockset-go-client"
	rockerr "github.com/rockset/rockset-go-client/errors"
	"github.com/rockset/rockset-go-client/openapi"
	"github.com/rockset/rockset-go-client/option"
)

type QueryModel struct {
	IntervalMs       uint64 `json:"intervalMs"`
	MaxDataPoints    int32  `json:"maxDataPoints"`
	QueryText        string `json:"queryText"`
	QueryParamStart  string `json:"queryParamStart"`
	QueryParamStop   string `json:"queryParamStop"`
	QueryTimeField   string `json:"queryTimeField"`
	QueryLabelColumn string `json:"queryLabelColumn"`
}

type Annotation struct {
	Datasource struct {
		Type string `json:"type"`
		Uid  string `json:"uid"`
	} `json:"datasource"`
	DatasourceId  int           `json:"datasourceId"`
	IntervalMs    int           `json:"intervalMs"`
	Limit         int           `json:"limit"`
	MatchAny      bool          `json:"matchAny"`
	MaxDataPoints int           `json:"maxDataPoints"`
	RefId         string        `json:"refId"`
	Tags          []interface{} `json:"tags"`
	Type          string        `json:"type"`
}

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
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
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	apiKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["apiKey"]
	if !found {
		return nil, fmt.Errorf("could not locate apiKey")
	}

	server, err := getServer(req.PluginContext.DataSourceInstanceSettings.JSONData)
	if err != nil {
		return nil, fmt.Errorf("could not locate server")
	}

	vi, err := getVI(req.PluginContext.DataSourceInstanceSettings.JSONData)
	if err != nil {
		return nil, fmt.Errorf("could not locate virtual instance id")
	}

	rs, err := rockset.NewClient(rockset.WithAPIKey(apiKey), rockset.WithAPIServer(server))
	if err != nil {
		return nil, fmt.Errorf("could create Rockset client: %w", err)
	}

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	log.DefaultLogger.Info("got queries", "count", len(req.Queries))
	for _, q := range req.Queries {
		res := Query(ctx, rs, vi, q)

		// save the response in a hashmap based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

// Query executes a single query and returns the result
func Query(ctx context.Context, rs Queryer, vi string, query backend.DataQuery) backend.DataResponse {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Error("recovered from panic", "error", r)
			log.DefaultLogger.Error(string(debug.Stack()))
		}
	}()
	log.DefaultLogger.Info("query", "refId", query.RefID, "JSON", string(query.JSON), "type", query.QueryType)

	var response backend.DataResponse

	// Unmarshal the request JSON into our QueryModel.
	var qm QueryModel
	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to unmarshal query: %v", err.Error()))
	}

	// set defaults and trim ":" from the start/stop
	if qm.QueryParamStart == "" {
		qm.QueryParamStart = "startTime"
	}
	qm.QueryParamStart = strings.TrimPrefix(qm.QueryParamStart, ":")
	if qm.QueryParamStop == "" {
		qm.QueryParamStop = "stopTime"
	}
	qm.QueryParamStop = strings.TrimPrefix(qm.QueryParamStop, ":")
	if qm.QueryTimeField == "" {
		qm.QueryTimeField = "_event_time"
		log.DefaultLogger.Info("setting default query time field", "value", qm.QueryTimeField)
	}

	log.DefaultLogger.Info("query",
		"interval", qm.IntervalMs,
		"max data points", qm.MaxDataPoints,
		"from", query.TimeRange.From,
		"to", query.TimeRange.To,
		"duration", query.TimeRange.To.Sub(query.TimeRange.From).String())

	options := []option.QueryOption{
		option.WithParameter(qm.QueryParamStart, "timestamp", query.TimeRange.From.UTC().Format(time.RFC3339)),
		option.WithParameter(qm.QueryParamStop, "timestamp", query.TimeRange.To.UTC().Format(time.RFC3339)),
		option.WithDefaultRowLimit(qm.MaxDataPoints),
	}
	if vi != "" {
		options = append(options, option.WithVirtualInstance(vi))
	}
	log.DefaultLogger.Info("query", "SQL", qm.QueryText, "vi", vi)

	qr, err := rs.Query(ctx, qm.QueryText, options...)
	if err != nil {
		var re rockerr.Error
		var errMessage string
		statusCode := backend.StatusUnknown
		if errors.As(err, &re) {
			statusCode = backend.Status(re.StatusCode)
			errMessage = fmt.Sprintf("There was a problem executing your query: Error ID [%s] - Query ID [%s]\nLine: %d Column: %d\n%s",
				re.GetErrorId(), re.GetQueryId(), re.GetLine(), re.GetColumn(), re.Error())
		} else {
			errMessage = fmt.Sprintf("There was a problem executing your query:\n%s", err.Error())
		}

		log.DefaultLogger.Error("query error", "error", errMessage)
		return backend.ErrDataResponse(statusCode, errMessage)
	}
	log.DefaultLogger.Info("query response", "elapsedTime", qr.Stats.ElapsedTimeMs, "# of results", len(qr.Results))

	if len(qr.Results) == 0 {
		return backend.ErrDataResponse(backend.StatusValidationFailed, "Query returned no rows")
	}
	// we don't allow SELECT *, as it doesn't set the ColumnFields
	if len(qr.ColumnFields) == 0 {
		return backend.ErrDataResponse(backend.StatusValidationFailed,
			"Query must not use 'SELECT *', instead explicitly specify the columns to return")
	}

	labels, err := extractLabelValues(qm.QueryLabelColumn, qr.Results)
	if err != nil {
		errMsg := fmt.Sprintf("label generation error: %s", err.Error())
		log.DefaultLogger.Error(errMsg)
		return backend.ErrDataResponse(backend.StatusInternal, errMsg)
	}
	log.DefaultLogger.Debug("extracted labels", "labels", labels)

	frame := makeFrame(qm.QueryText, qr)

	for _, label := range labels {
		for i, c := range qr.ColumnFields {
			// skip the time field and the label column
			if c.Name == qm.QueryTimeField || c.Name == qm.QueryLabelColumn {
				continue
			}
			log.DefaultLogger.Debug("column", "i", i, "name", c.Name, "label", label)

			fields, err := extractFields(qm.QueryTimeField, c.Name, qm.QueryLabelColumn, label, qr.Results)
			if err != nil {
				errMsg := fmt.Sprintf("failed to create frame for %s: %v", c.Name, err)
				return backend.ErrDataResponse(backend.StatusUnknown, errMsg)
			}

			log.DefaultLogger.Info("adding fields", "fields", len(fields))
			frame.Fields = append(frame.Fields, fields...)
			response.Frames = append(response.Frames, frame)
		}
	}

	response.Frames = append(response.Frames, frame)
	return response
}

func makeFrame(query string, qr openapi.QueryResponse) *data.Frame {
	meta := data.FrameMeta{
		Type:                data.FrameTypeTimeSeriesWide,
		TypeVersion:         data.FrameTypeVersion{0, 1},
		ExecutedQueryString: query,
		Stats: []data.QueryStat{
			{
				FieldConfig: data.FieldConfig{DisplayName: "query time", Unit: "ms"},
				Value:       float64(qr.Stats.GetElapsedTimeMs()),
			},
			{
				FieldConfig: data.FieldConfig{DisplayName: "throttled time", Unit: "µs"},
				Value:       float64(qr.Stats.GetThrottledTimeMicros()),
			},
			{
				FieldConfig: data.FieldConfig{DisplayName: "documents in the result"},
				Value:       float64(qr.GetResultsTotalDocCount()),
			},
		},
	}

	if qr.HasQueryErrors() {
		fields := make([]string, len(qr.GetQueryErrors()))
		for i, e := range qr.GetQueryErrors() {
			fields[i] = e.GetMessage()
		}
		meta.Notices = append(meta.Notices, data.Notice{
			Severity: data.NoticeSeverityError,
			Text:     strings.Join(fields, ", "),
		})
	}
	if qr.HasWarnings() {
		meta.Notices = append(meta.Notices, data.Notice{
			Severity: data.NoticeSeverityWarning,
			Text:     strings.Join(qr.GetWarnings(), ", "),
		})
	}

	frame := data.NewFrame("response")
	frame.SetMeta(&meta)

	return frame
}

// extracts fields in wide format
// https://grafana.com/developers/plugin-tools/introduction/data-frames#wide-format
func extractFields(timeField, valueField, labelColumn, label string, qr []map[string]interface{}) ([]*data.Field, error) {
	var times []time.Time
	var floatValues []float64
	var boolValues []bool
	var stringValues []string

	var labels map[string]string
	// empty label means there is no label column and labels should use the zero value, which is nil
	if labelColumn != "" {
		labels = map[string]string{labelColumn: label}
	}

	// iterate over the rows
	for _, row := range qr {
		if labelColumn != "" {
			if l, found := row[labelColumn]; found && l != label {
				// skip rows which doesn't match the label
				log.DefaultLogger.Debug("skipping row due to missing label", "label", label)
				continue
			}
		}

		// the value might not be present in every row
		v, found := row[valueField]
		if !found {
			log.DefaultLogger.Debug("skipping row due to missing value", "value", valueField)
			continue
		}

		switch v.(type) {
		case bool:
			boolValues = append(boolValues, v.(bool))
		case string:
			stringValues = append(stringValues, v.(string))
		case float64:
			floatValues = append(floatValues, v.(float64))
		default:
			if v == "" {
			}
			log.DefaultLogger.Error("unknown type", "type", fmt.Sprintf("%T", v), "value", v, "column", valueField)
			continue
		}

		t, err := parseTime(timeField, row)
		if err != nil {
			return nil, err
		}
		times = append(times, t)
	}

	// add the time dimension
	fields := []*data.Field{data.NewField("time", labels, times)}
	// TODO should we add all fields > 0? or should we enfoce that all values in a field are of the same type?
	if len(floatValues) > 0 {
		fields = append(fields, data.NewField(valueField, labels, floatValues))
	} else if len(stringValues) > 0 {
		fields = append(fields, data.NewField(valueField, labels, stringValues))
	} else if len(boolValues) > 0 {
		fields = append(fields, data.NewField(valueField, labels, boolValues))
	} else {
		return nil, fmt.Errorf("failed to create frame for %s: no values found", valueField)
	}

	return fields, nil
}

// extract the set of label values from the label column
func extractLabelValues(labelColumn string, results []map[string]interface{}) ([]string, error) {
	labels := make([]string, 0)
	seen := make(map[string]struct{})

	// if there isn't any label column specified, add an empty string, so we can use it as a special case
	if labelColumn == "" {
		return []string{""}, nil
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

		if _, found = seen[l]; !found {
			labels = append(labels, l)
			seen[l] = struct{}{}
		}
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("label column '%s' doesn't contain any string values", labelColumn)
	}

	return labels, nil
}

func parseTime(key string, fields map[string]interface{}) (time.Time, error) {
	value, ok := fields[key]
	if !ok {
		// TODO include a list of available columns
		return time.Time{}, fmt.Errorf("could not find column %s in query response", key)
	}

	v, ok := value.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("could not cast %s (%v) %T to string", key, value, value)
	}

	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to convert %s to time: %w", v, err)
	}

	return t, nil
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Debug("CheckHealth called")

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

	// This call requires the GET_ORG_GLOBAL permission, which we can't rely on being granted,
	// so perhaps we should use `SELECT 1` instead? At least the error message from the call
	// highlight that the permission is missing.

	// validate that we can connect by getting the org info
	org, err := rs.GetOrganization(ctx)
	if err != nil {
		log.DefaultLogger.Error("CheckHealth failed", "err", err)
		return healthError("failed get connect to Rockset: %s", err.Error()), nil
	}
	log.DefaultLogger.Debug("CheckHealth successful", "org", org.GetId())

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: fmt.Sprintf("Rockset datasource is working, connected to %s", org.GetId()),
	}, nil
}

func getServer(data []byte) (string, error) {
	var conf struct {
		Server string `json:"server"`
	}

	if err := json.Unmarshal(data, &conf); err != nil {
		return "", fmt.Errorf("failed to unmarshal server json: %w", err)
	}

	return conf.Server, nil
}

func getVI(data []byte) (string, error) {
	var conf struct {
		VI string `json:"vi"`
	}

	if err := json.Unmarshal(data, &conf); err != nil {
		return "", fmt.Errorf("failed to unmarshal server json: %w", err)
	}

	return conf.VI, nil
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
