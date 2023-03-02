import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { VariableQueryEditor } from 'components/VariableQueryEditor';
import { RocksetQuery, RocksetDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, RocksetQuery, RocksetDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor)
  .setVariableQueryEditor(VariableQueryEditor); // TODO: setVariableQueryEditor is deprecated, but undocumented https://github.com/grafana/grafana/issues/63619
