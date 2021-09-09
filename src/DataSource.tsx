import {
  DataSourceInstanceSettings,
  DataQueryRequest,
  DataQueryResponse,
  MetricFindValue,
  DataFrame,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { RocksetDataSourceOptions, RocksetQuery, MyVariableQuery } from './types';

export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
    super(instanceSettings);
  }

  async metricFindQuery?(query: MyVariableQuery, options?: any): Promise<MetricFindValue[]> {
    const request = {
      targets: [
        {
          queryText: query.rawQuery,
          queryTimeField: '_event_time',
          queryValueField: 'value',
          refId: 'metricFindQuery',
        },
      ],
    } as DataQueryRequest<MyVariableQuery>;

    let res: DataQueryResponse;

    try {
      res = await this.query(request).toPromise();
    } catch (err) {
      return Promise.reject(err);
    }

    if (!res.data.length || !res.data[0].fields.length) {
      return [];
    }

    return (res.data[0] as DataFrame).fields[1].values.toArray().map(_ => ({ text: _.toString() }));
  }

  applyTemplateVariables(query: RocksetQuery) {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      queryText: query.queryText ? templateSrv.replace(query.queryText) : '',
    };
  }
}
