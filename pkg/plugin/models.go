package plugin

type queryModel interface {
	GetQueryParamStart() string
	GetQueryParamStop() string
	GetIntervalMs() uint64
	GetMaxDataPoints() int32
}

type DatasourceModel struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

type BaseQueryModel struct {
	Datasource   Datasource `json:"datasource"`
	RefID        string     `json:"refId"`
	DatasourceID int32      `json:"datasourceId"`
	IntervalMs   uint64     `json:"intervalMs"`
}

type QueryModel struct {
	BaseQueryModel
	QueryParamStart string `json:"queryParamStart"`
	QueryParamStop  string `json:"queryParamStop"`
	QueryTimeField  string `json:"queryTimeField"`
	MaxDataPoints   int32  `json:"maxDataPoints"`
	QueryText       string `json:"queryText"`
}

func (q QueryModel) GetQueryParamStart() string { return q.QueryParamStart }
func (q QueryModel) GetQueryParamStop() string  { return q.QueryParamStop }
func (q QueryModel) GetIntervalMs() uint64      { return q.IntervalMs }
func (q QueryModel) GetMaxDataPoints() int32    { return q.MaxDataPoints }

type MetricsQueryModel struct {
	QueryModel
	QueryLabelColumn string `json:"queryLabelColumn"`
}

type AnnotationsQueryModel struct {
	QueryModel
}

type VariablesQueryModel struct {
	QueryModel
}
