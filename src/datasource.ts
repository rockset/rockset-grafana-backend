import {CoreApp, DataSourceInstanceSettings, ScopedVars} from '@grafana/data';
import {DataSourceWithBackend, getTemplateSrv} from '@grafana/runtime';
import {AnnotationEditor} from './components/AnnotationEditor';

import {DEFAULT_QUERY, RocksetDataSourceOptions, RocksetQuery} from './types';


export class DataSource extends DataSourceWithBackend<RocksetQuery, RocksetDataSourceOptions> {
    constructor(instanceSettings: DataSourceInstanceSettings<RocksetDataSourceOptions>) {
        super(instanceSettings);


        this.annotations = {
            QueryEditor: AnnotationEditor,
        }
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
