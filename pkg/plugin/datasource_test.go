package plugin_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/rockset/rockset-go-client/openapi"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/assert"

	"github.com/rockset/rockset-grafana-backend/pkg/plugin"
	"github.com/rockset/rockset-grafana-backend/pkg/plugin/fake"
)

func TestQueryData(t *testing.T) {
	ds := plugin.Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

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

func TestQuery(t *testing.T) {
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

	rs := &fake.FakeQueryer{}
	rs.QueryReturns(qr, nil)

	qm := plugin.MetricsQueryModel{
		QueryModel: plugin.QueryModel{
			QueryTimeField: "time",
		},
	}

	resp := plugin.MetricsQuery(
		context.Background(),
		rs,
		"",
		backend.DataQuery{
			RefID: "A",
			JSON:  marshal(t, qm),
		},
	)

	require.Nil(t, resp.Error)
	require.Len(t, resp.Frames, 1, "frames")
	require.Len(t, resp.Frames[0].Fields, len(qr.ColumnFields), "fields")

	// verify that the Frame values match the QueryResponse
	for i, column := range resp.Frames[0].Fields {
		for j := 0; j < column.Len(); j++ {
			switch i {
			case 0: // time
			case 1:
				f, err := column.NullableFloatAt(j)
				assert.NoError(t, err)
				assert.Equal(t, data[j].V1, *f)
			case 2:
				f, err := column.NullableFloatAt(j)
				assert.NoError(t, err)
				assert.Equal(t, data[j].V2, int(*f))
			case 3:
				b, ok := column.At(j).(*bool)
				assert.True(t, ok)
				assert.Equal(t, data[j].V3, *b)
			case 4:
				s, ok := column.At(j).(*string)
				assert.True(t, ok)
				assert.Equal(t, data[j].V4, *s)
			default:
				// do nothing
				t.Logf("i: %d, j: %d", i, j)
			}

		}
	}
}

func marshal(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.Nil(t, err)

	return b
}
