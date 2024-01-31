import {DataSourceJsonData} from '@grafana/data';
import {DataQuery} from '@grafana/schema';

export interface RocksetQuery extends DataQuery {
    queryText?: string;
    queryParamStart: string;
    queryParamStop: string;
    queryTimeField: string;
    queryLabelColumn: string;
}

export const DEFAULT_QUERY: Partial<RocksetQuery> = {
    queryText: `-- sample metrics query
SELECT
  TIME_BUCKET(MINUTES(5), e._event_time) AS _event_time,
  COUNT(e.type) AS count,
  e.kind AS label
FROM
  commons._events e
-- you MUST specify a WHERE clause which scopes the query using :startTime and :stopTime
-- as the Rockset plugin executes the query with these parameters
WHERE
  e._event_time > :startTime AND
  e._event_time < :stopTime
GROUP BY
  _event_time,
  label
ORDER BY
  _event_time`,
    queryParamStart: ':startTime',
    queryParamStop: ':stopTime',
    queryTimeField: '_event_time',
    queryLabelColumn: 'label',
};

/**
 * These are options configured for each DataSource instance
 */
export interface RocksetDataSourceOptions extends DataSourceJsonData {
    server?: string;
    vi?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface RocksetSecureJsonData {
    apiKey?: string;
}

export interface RocksetVariableQuery {
    namespace: string;
    rawQuery: string;
}
