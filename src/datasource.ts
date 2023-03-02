import {
  DataSourceInstanceSettings,
  ScopedVars
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { RocksetQuery, RocksetDataSourceOptions } from './types';

export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
    super(instanceSettings);
  }

  applyTemplateVariables(query: RocksetQuery, scopedVars: ScopedVars): Record<string, any> {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      queryText: query.queryText ? templateSrv.replace(query.queryText, scopedVars) : '',
    };
  }
}
