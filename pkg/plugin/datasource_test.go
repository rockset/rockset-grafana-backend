package plugin_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/rockset/rockset-go-client"
	"github.com/rockset/rockset-go-client/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockset/rockset-grafana-backend/pkg/plugin"
	"github.com/rockset/rockset-grafana-backend/pkg/plugin/fake"
)

type testType struct {
	Time string  `json:"time"`
	V1   float64 `json:"v1"`
	V2   int     `json:"v2"`
	V3   bool    `json:"v3"`
	V4   string  `json:"v4"`
}

func prepareTestData(t *testing.T, in []testType) []map[string]interface{} {
	t.Helper()

	x, err := json.Marshal(in)
	require.NoError(t, err)

	var y []map[string]interface{}
	err = json.Unmarshal(x, &y)
	require.NoError(t, err)

	return y
}

func TestQueryData(t *testing.T) {
	data := []testType{
		{Time: "2024-01-23T19:25:17.000000-08:00", V1: 1.111, V2: 1, V3: true, V4: "foo"},
		{Time: "2024-01-23T19:25:17.000000-08:00", V1: 2.222, V2: 2, V3: false, V4: "bar"},
		{Time: "2024-01-23T19:26:17.000000-08:00", V1: 3.333, V2: 3, V3: false, V4: "foo"},
		{Time: "2024-01-23T19:26:17.000000-08:00", V1: 4.444, V2: 4, V3: true, V4: "bar"},
	}
	qr := openapi.QueryResponse{
		Results:      prepareTestData(t, data),
		ColumnFields: []openapi.QueryFieldType{{Name: "time"}, {Name: "v1"}, {Name: "v2"}, {Name: "v3"}, {Name: "v4"}},
		Stats:        &openapi.QueryResponseStats{},
	}
	require.Equal(t, len(qr.Results[0]), len(qr.ColumnFields))

	rc := fake.FakeRockClient{}
	rc.QueryReturns(qr, nil)

	ds := plugin.RocksetDatasource{
		ClientFactory: func(option ...rockset.RockOption) (plugin.RockClient, error) {
			return &rc, nil
		},
	}

	qm := plugin.MetricsQueryModel{
		QueryModel: plugin.QueryModel{
			QueryTimeField:  "time",
			QueryParamStart: "2024-01-23T19:25:00.000000-08:00",
			QueryParamStop:  "2024-01-23T19:27:00.000000-08:00",
		},
		QueryLabelColumn: "v4",
	}

	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		PluginContext: fakePluginContext(),
		Queries: []backend.DataQuery{
			backend.DataQuery{
				RefID: "A",
				JSON:  marshal(t, qm),
			},
		},
	})
	require.NoError(t, err)

	require.Len(t, resp.Responses, 1)
	frames := resp.Responses["A"].Frames

	require.Len(t, frames, 2, "frames")
	require.Len(t, frames[0].Fields, len(qr.ColumnFields)-1, "fields")
	require.Len(t, frames[1].Fields, len(qr.ColumnFields)-1, "fields")

	// verify that the Frame values match the QueryResponse
	var tm time.Time
	assert.Equal(t, frames[0].Fields[0].Name, "time")

	tm = frames[0].Fields[0].At(0).(time.Time)
	assert.Equal(t, tm.UTC().String(), "2024-01-24 03:25:17 +0000 UTC")
	tm = frames[0].Fields[0].At(1).(time.Time)
	assert.Equal(t, tm.UTC().String(), "2024-01-24 03:26:17 +0000 UTC")

	assert.Equal(t, frames[0].Fields[1].Name, "v1")
	assert.Contains(t, frames[0].Fields[1].Labels, "v4")
	assert.Contains(t, frames[0].Fields[1].Labels["v4"], "foo")
	f, err := frames[0].Fields[1].FloatAt(0)
	require.NoError(t, err)
	assert.Equal(t, f, 1.111)
	f, err = frames[0].Fields[1].FloatAt(1)
	require.NoError(t, err)
	assert.Equal(t, f, 3.333)
}

func marshal(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.Nil(t, err)

	return b
}

func TestHealthCheck(t *testing.T) {
	ctx := context.TODO()
	f := fake.FakeRockClient{}
	f.GetOrganizationReturns(openapi.Organization{
		Id: openapi.PtrString("org"),
	}, nil)
	ds := plugin.RocksetDatasource{
		ClientFactory: func(option ...rockset.RockOption) (plugin.RockClient, error) {
			return &f, nil
		},
	}

	resp, err := ds.CheckHealth(ctx, &backend.CheckHealthRequest{
		PluginContext: fakePluginContext(),
	})
	require.NoError(t, err)
	assert.Equal(t, resp.Status, backend.HealthStatusOk)
	assert.Equal(t, "Rockset datasource is working, connected to org", resp.Message)
}

func fakePluginContext() backend.PluginContext {
	return backend.PluginContext{
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
			DecryptedSecureJSONData: map[string]string{"apiKey": "foobar"},
			JSONData:                []byte(`{"server":"api.usw2a1.rockset.com","vi":"vi"}`),
		},
	}
}
