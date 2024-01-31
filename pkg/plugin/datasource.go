package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
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

// Make sure RocksetDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*RocksetDatasource)(nil)
	_ backend.CheckHealthHandler    = (*RocksetDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*RocksetDatasource)(nil)
)

// NewRocksetDatasource creates a new datasource instance.
func NewRocksetDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &RocksetDatasource{
		RockFactory,
	}, nil
}

// RocksetDatasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type RocksetDatasource struct {
	ClientFactory func(...rockset.RockOption) (RockClient, error) `json:"-"`
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *RocksetDatasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *RocksetDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
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

	rs, err := d.ClientFactory(rockset.WithAPIKey(apiKey), rockset.WithAPIServer(server),
		rockset.WithCustomHeader("rockset-grafana-backend", "v0.3"))
	if err != nil {
		id := "unknown"
		if len(req.Queries) > 0 {
			id = req.Queries[0].RefID
		}

		return &backend.QueryDataResponse{
			Responses: map[string]backend.DataResponse{
				id: backend.ErrDataResponse(backend.StatusUnknown,
					fmt.Sprintf("could create Rockset datasource: %v", err)),
			},
		}, nil
	}

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	log.DefaultLogger.Info("got queries", "count", len(req.Queries))
	for _, q := range req.Queries {
		log.DefaultLogger.Info("query", "refId", q.RefID, "JSON", string(q.JSON))

		var res backend.DataResponse
		switch q.RefID {
		case "Anno":
			res = AnnotationsQuery(ctx, rs, vi, q)
		case "variable-query":
			res = VariablesQuery(ctx, rs, vi, q)
		default:
			res = MetricsQuery(ctx, rs, vi, q)
		}

		// save the response in a hashmap based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

// AnnotationsQuery handles annotation queries from grafana
func AnnotationsQuery(ctx context.Context, rs RockClient, vi string, query backend.DataQuery) (response backend.DataResponse) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Error("recovered from panic", "error", r)
			log.DefaultLogger.Error(string(debug.Stack()))

			response.Error = fmt.Errorf("internal plugin error, please contact Rockset support")
			response.Status = backend.StatusInternal
		}
	}()

	var qm AnnotationsQueryModel
	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to unmarshal query: %v", err.Error()))
	}

	options := buildQueryOptions(qm, query.TimeRange.From, query.TimeRange.To, vi)
	log.DefaultLogger.Info("executing annotations query", "SQL", qm.QueryText)
	qr, err := rs.Query(ctx, qm.QueryText, options...)
	if err != nil {
		return errorToResponse(err)
	}
	logQueryResponse(qr)

	if len(qr.ColumnFields) == 0 {
		return backend.ErrDataResponse(backend.StatusValidationFailed,
			"Query must not use 'SELECT *', instead explicitly specify the columns to return")
	}

	frame := makeFrame("annotations", qm.QueryText, qr)
	fields, err := extractWideFields(qm.QueryTimeField, "", "", qr)
	if err != nil {
		errMsg := fmt.Sprintf("failed to extract fields: %v", err)
		return backend.ErrDataResponse(backend.StatusUnknown, errMsg)
	}

	frame.Fields = append(frame.Fields, fields...)
	response.Frames = append(response.Frames, frame)

	return response
}

// VariablesQuery returns list of values for template variables
func VariablesQuery(ctx context.Context, rs RockClient, vi string, query backend.DataQuery) (response backend.DataResponse) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Error("recovered from panic", "error", r)
			log.DefaultLogger.Error(string(debug.Stack()))

			response.Error = fmt.Errorf("internal plugin error, please contact Rockset support")
			response.Status = backend.StatusInternal
		}
	}()

	var qm VariablesQueryModel
	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to unmarshal query: %v", err.Error()))
	}

	var options []option.QueryOption
	if vi != "" {
		options = append(options, option.WithVirtualInstance(vi))
	}

	log.DefaultLogger.Info("executing variables query", "SQL", qm.QueryText)
	qr, err := rs.Query(ctx, qm.QueryText, options...)
	if err != nil {
		return errorToResponse(err)
	}
	logQueryResponse(qr)

	frame := makeFrame("variables", qm.QueryText, qr)
	field, err := extractVariableField(qr.Results)
	if err != nil {
		return errorToResponse(err)
	}

	frame.Fields = append(frame.Fields, field...)
	response.Frames = append(response.Frames, frame)

	return response
}

// MetricsQuery executes a single query and returns the result
func MetricsQuery(ctx context.Context, rs RockClient, vi string, query backend.DataQuery) backend.DataResponse {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Error("recovered from panic", "error", r)
			log.DefaultLogger.Error(string(debug.Stack()))
		}
	}()

	var response backend.DataResponse

	// Unmarshal the request JSON into our MetricsQueryModel.
	var qm MetricsQueryModel
	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to unmarshal query: %v", err.Error()))
	}

	options := buildQueryOptions(qm, query.TimeRange.From, query.TimeRange.To, vi)
	log.DefaultLogger.Info("executing metrics query", "SQL", qm.QueryText)

	qr, err := rs.Query(ctx, qm.QueryText, options...)
	if err != nil {
		return errorToResponse(err)
	}
	logQueryResponse(qr)

	if len(qr.Results) == 0 {
		return backend.ErrDataResponse(backend.StatusValidationFailed, "Query returned no rows")
	}
	// we don't allow SELECT *, as it doesn't set the ColumnFields, but we could calculate that here
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

	for _, label := range labels {
		log.DefaultLogger.Info("processing label", "label", label)
		frame := makeFrame("metrics", qm.QueryText, qr)

		fields, err := extractWideFields(qm.QueryTimeField, qm.QueryLabelColumn, label, qr)
		if err != nil {
			errMsg := fmt.Sprintf("failed to extract fields for label %s: %v", label, err)
			return backend.ErrDataResponse(backend.StatusUnknown, errMsg)
		}

		log.DefaultLogger.Info("adding frame", "fields", len(fields), "label", label)
		frame.Fields = append(frame.Fields, fields...)
		response.Frames = append(response.Frames, frame)
	}

	return response
}

func buildQueryOptions[T queryModel](qm T, from, to time.Time, vi string) []option.QueryOption {
	var options []option.QueryOption
	var opts []any

	log.DefaultLogger.Info("query parameters",
		"interval", qm.GetIntervalMs(),
		"max data points", qm.GetMaxDataPoints(),
		"from", from,
		"to", to,
		"duration", to.Sub(from).String())

	if qm.GetIntervalMs() > 0 {
		opts = append(opts, ":interval", qm.GetIntervalMs())
		options = append(options, option.WithParameter("interval", "int", strconv.FormatUint(qm.GetIntervalMs(), 10)))
	}

	// set defaults and trim ":" from the start/stop
	start := strings.TrimPrefix(qm.GetQueryParamStart(), ":")
	if start != "" {
		opts = append(opts, ":"+start, from.UTC().Format(time.RFC3339)) // ":" just for logging
		options = append(options, option.WithParameter(start, "timestamp", from.UTC().Format(time.RFC3339)))
	}

	stop := strings.TrimPrefix(qm.GetQueryParamStop(), ":")
	if stop != "" {
		opts = append(opts, ":"+stop, to.UTC().Format(time.RFC3339)) // ":" just for logging
		options = append(options, option.WithParameter(stop, "timestamp", to.UTC().Format(time.RFC3339)))
	}

	if qm.GetMaxDataPoints() > 0 {
		opts = append(opts, "row limit", qm.GetMaxDataPoints())
		options = append(options, option.WithDefaultRowLimit(qm.GetMaxDataPoints()))
	}

	if vi != "" {
		opts = append(opts, "vi", vi)
		options = append(options, option.WithVirtualInstance(vi))
	}

	log.DefaultLogger.Info("query options", opts...)

	return options
}

func errorToResponse(err error) backend.DataResponse {
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

func logQueryResponse(qr openapi.QueryResponse) {
	page := qr.GetPagination()
	log.DefaultLogger.Info("query response",
		"elapsedTime", qr.Stats.ElapsedTimeMs,
		"docs", len(qr.Results),
		"errors", qr.GetQueryErrors(),
		"warnings", strings.Join(qr.GetWarnings(), ", "),
		"queryID", qr.GetQueryId(),
		"pagination", page.GetNextCursor(),
	)
}

func makeFrame(name, query string, qr openapi.QueryResponse) *data.Frame {
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
				Value:       float64(len(qr.Results)),
			},
		},
	}

	page := qr.GetPagination()
	if page.GetNextCursor() != "" {
		meta.Notices = append(meta.Notices, data.Notice{
			Severity: data.NoticeSeverityWarning,
			Text:     "pagination needed",
		})
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

	frame := data.NewFrame(name)
	frame.SetMeta(&meta)

	return frame
}

func extractVariableField(qr []map[string]interface{}) ([]*data.Field, error) {
	if len(qr) == 0 {
		return nil, fmt.Errorf("got empty query response")
	}
	if len(qr[0]) != 1 {
		return nil, fmt.Errorf("expected exactly one column, got %d", len(qr))
	}
	var variables []string
	seen := make(map[string]struct{})

	for _, row := range qr {
		for _, v := range row {
			if s, ok := v.(string); ok {
				if _, found := seen[s]; !found {
					variables = append(variables, s)
					seen[s] = struct{}{}
				}
			}
		}
	}
	field := data.NewField("variables", nil, variables)

	return []*data.Field{field}, nil
}

func firstValueInColumn(name string, qr []map[string]interface{}) interface{} {
	for _, row := range qr {
		if v, found := row[name]; found && v != nil {
			return v
		}
	}

	return nil
}

const DefaultTimeColumn = "_event_time"

// extracts fields in wide format
// https://grafana.com/developers/plugin-tools/introduction/data-frames#wide-format
func extractWideFields(timeColumn, labelColumn, label string, qr openapi.QueryResponse) ([]*data.Field, error) {
	var fields []*data.Field

	// the annotation query doesn't set the time column, unless changed,
	if timeColumn == "" {
		timeColumn = DefaultTimeColumn
	}

	// iterate over the columns, extracting each into a field for the frame
	for i, c := range qr.ColumnFields {
		// skip label column
		if c.Name == labelColumn {
			log.DefaultLogger.Debug("skipping column", "i", i, "name", c.Name)
			continue
		}
		if c.Name == timeColumn {
			times, err := extractTimeColumn(timeColumn, label, labelColumn, qr.Results)
			if err != nil {
				return nil, err
			}
			fields = append(fields, data.NewField("time", nil, times))
			continue
		}
		log.DefaultLogger.Debug("processing column", "i", i, "name", c.Name, "label", label)

		// get the first value of the column and assume it is what every value will be the same type
		t := firstValueInColumn(c.Name, qr.Results)

		// extract the values from the column
		switch t.(type) {
		case bool:
			fields = append(fields, data.NewField(c.Name, data.Labels{labelColumn: label},
				extractColumnValues[bool](c.Name, label, labelColumn, qr.Results)))
		case string:
			fields = append(fields, data.NewField(c.Name, data.Labels{labelColumn: label},
				extractColumnValues[string](c.Name, label, labelColumn, qr.Results)))
		case float64:
			fields = append(fields, data.NewField(c.Name, data.Labels{labelColumn: label},
				extractColumnValues[float64](c.Name, label, labelColumn, qr.Results)))
		default:
			log.DefaultLogger.Error("unknown type", "type", fmt.Sprintf("%T", t), "value", t)
		}
	}

	return fields, nil
}

func extractTimeColumn(name, label, labelColumn string, qr []map[string]interface{}) ([]time.Time, error) {
	var times []time.Time

	for _, row := range qr {
		if labelColumn != "" {
			if l, found := row[labelColumn]; found && l != label {
				log.DefaultLogger.Debug("skipping column", "name", name)
				continue
			}
		}

		value, found := row[name]
		if !found {
			return times, fmt.Errorf("time column not found: %s", name)
		}

		switch value.(type) {
		case string:
			t, err := time.Parse(time.RFC3339Nano, value.(string))
			if err != nil {
				return nil, fmt.Errorf("failed to convert %s to time: %w", value, err)
			}
			times = append(times, t)
		default:
			return nil, fmt.Errorf("column %s is of type '%T', not the expected type 'string'", name, value)
		}
	}

	return times, nil
}

func extractColumnValues[T any](name, label, labelColumn string, qr []map[string]interface{}) []*T {
	var column []*T
	for i, row := range qr {
		if labelColumn != "" {
			if l, found := row[labelColumn]; found && l != label {
				// skip rows which doesn't match the label
				log.DefaultLogger.Debug("skipping row due to missing label", "want", label, "is", l)
				continue
			}
		}

		value, found := row[name]
		if !found {
			log.DefaultLogger.Error("column not found", "column", name, "i", i)
			column = append(column, nil)
			continue
		}

		switch value.(type) {
		case T:
			v := value.(T)
			column = append(column, &v)
		default:
			log.DefaultLogger.Error("column is not of type",
				"column", name, "i", i, "type", fmt.Sprintf("%T", value), "value", value)
			column = append(column, nil)
		}
	}

	return column
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
			log.DefaultLogger.Debug("could not lookup label", "column", labelColumn)
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
		log.DefaultLogger.Warn("label column doesn't contain any string values", "column", labelColumn)
		return []string{""}, nil
	}

	return labels, nil
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *RocksetDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Debug("CheckHealth called")

	apiKey, found := req.PluginContext.DataSourceInstanceSettings.DecryptedSecureJSONData["apiKey"]
	if !found {
		return healthError("failed to get api key"), nil
	}

	server, err := getServer(req.PluginContext.DataSourceInstanceSettings.JSONData)
	if err != nil {
		return healthError("unable to unmarshal json"), nil
	}

	rs, err := d.ClientFactory(rockset.WithAPIKey(apiKey), rockset.WithAPIServer(server))
	rockset.NewClient(rockset.WithAPIKey(apiKey), rockset.WithAPIServer(server))
	if err != nil {
		return healthError("failed to create Rockset client: %s", err.Error()), nil
	}

	// This call requires the GET_ORG_GLOBAL permission, which we can't rely on being granted,
	// so perhaps we should use `SELECT 1` instead? At least the error message from the call
	// highlight that the permission is missing.

	// validate that we can connect by getting the org info
	org, err := rs.GetOrganization(ctx)
	if err != nil {
		log.DefaultLogger.Error("CheckHealth failed", "err", err.Error())
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
