import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { RocksetDataSourceOptions, RocksetQuery } from './types';

export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
    super(instanceSettings);
  }
}
