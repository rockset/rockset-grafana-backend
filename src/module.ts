import {DataSourcePlugin} from '@grafana/data';
import {DataSource} from './datasource';
import {ConfigEditor} from './components/ConfigEditor';
import {QueryEditor} from './components/QueryEditor';
import {VariableQueryEditor} from 'components/VariableQueryEditor';
import {RocksetDataSourceOptions, RocksetQuery} from './types';

export const plugin = new DataSourcePlugin<DataSource, RocksetQuery, RocksetDataSourceOptions>(DataSource)
    .setConfigEditor(ConfigEditor)
    .setQueryEditor(QueryEditor)
    // TODO this should be replaced by DataSourceVariableSupport
    // .setVariableQueryEditor(VariableQueryEditor)
;
