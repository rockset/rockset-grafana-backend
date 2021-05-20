import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { RocksetDataSourceOptions, RocksetQuery } from './types';
import { getTemplateSrv } from '@grafana/runtime';

export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
    super(instanceSettings);
  }
  applyTemplateVariables(query: RocksetQuery) {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      queryText: query.queryText ? templateSrv.replace(query.queryText) : '',
    };
  }
}
