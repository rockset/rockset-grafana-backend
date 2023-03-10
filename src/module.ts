import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/configEditor';
import { QueryEditor } from './components/queryEditor';
import { VariableQueryEditor } from 'components/variableQueryEditor';
import { RocksetQuery, RocksetDataSourceOptions } from './types';

// TODO: setVariableQueryEditor is deprecated, but alternatives are marked alpha
//  and undocumented https://github.com/grafana/grafana/issues/63619
export const plugin = new DataSourcePlugin<DataSource, RocksetQuery, RocksetDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor)
  .setVariableQueryEditor(VariableQueryEditor);
