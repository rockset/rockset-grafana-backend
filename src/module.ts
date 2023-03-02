import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { VariableQueryEditor } from 'components/VariableQueryEditor';
import { RocksetQuery, RocksetDataSourceOptions } from './types';

// TODO: setVariableQueryEditor is deprecated, but alternatives are marked alpha
//  and undocumented https://github.com/grafana/grafana/issues/63619
export const plugin = new DataSourcePlugin<DataSource, RocksetQuery, RocksetDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor)
  .setVariableQueryEditor(VariableQueryEditor);
