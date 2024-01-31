import {
    CoreApp,
    DataQueryRequest,
    DataSourceInstanceSettings,
    MetricFindValue,
    ScopedVars,
    VariableSupportType
} from '@grafana/data';
import {DataSourceWithBackend, getTemplateSrv} from '@grafana/runtime';
import {AnnotationEditor} from './components/AnnotationEditor';
import {VariableQueryEditor} from './components/VariableQueryEditor';

import {DEFAULT_QUERY, RocksetDataSourceOptions, RocksetQuery} from './types';


export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
    constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
        super(instanceSettings);


        this.annotations = {
            QueryEditor: AnnotationEditor,
            prepareQuery: (anno) => anno.target
        }

        this.variables = {
            editor: VariableQueryEditor,
            getType: () => VariableSupportType.Custom,
            query: (q: DataQueryRequest<RocksetQuery>) => this.query({
                ...q,
                targets: q.targets.map((t) => ({...t, refId: "variable-query"}))
            })
        }
    }

    async metricFindQuery(query: any, options?: any): Promise<MetricFindValue[]> {
        console.log({query, options});
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
