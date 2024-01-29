import {CoreApp, DataSourceInstanceSettings, MetricFindValue, ScopedVars, VariableSupportType} from '@grafana/data';
import {DataSourceWithBackend, getTemplateSrv} from '@grafana/runtime';
import {AnnotationEditor} from './components/AnnotationEditor';

import {DEFAULT_QUERY, RocksetDataSourceOptions, RocksetQuery} from './types';


export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
    constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
        super(instanceSettings);


        this.annotations = {
            QueryEditor: AnnotationEditor,
            prepareQuery: (anno) => anno.target
        }

        this.variables = {
            // TODO set VariableQueryEditor as the editor
            getType: () => VariableSupportType.Datasource,
            getDefaultQuery: () => ({
                queryTimeField: "_event_time",
                queryLabelColumn: "label",
                queryText: `SELECT
     'test' AS label,
     CURRENT_TIMESTAMP() AS _event_time
 `.trim()
            })
        }
    }

    async metricFindQuery(query: any, options?: any): Promise<MetricFindValue[]> {
        console.log({ query, options });
        // TODO;
        return [{
            text: "__TEST__"
        }]
    }

    applyTemplateVariables(query: RocksetQuery, scopedVars: ScopedVars): Record<string, any> {
        const templateSrv = getTemplateSrv();
        return {
            ...query,
            queryText: query.queryText ? templateSrv.replace(query.queryText, scopedVars) : '',
        };
    }

    getDefaultQuery(_: CoreApp): Partial<RocksetQuery> {
        return DEFAULT_QUERY;
    }
}
