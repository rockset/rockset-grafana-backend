import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface RocksetQuery extends DataQuery {
  queryText?: string;
  queryParamStart: string;
  queryParamStop: string;
  queryTimeField: string;
  queryValueField: string;
  queryLabelColumn: string;
}

export const defaultQuery: Partial<RocksetQuery> = {
  queryText: `-- sample query
SELECT
  TIME_BUCKET(MINUTES(5), _events._event_time) AS _event_time,
  COUNT(_events.type) AS value
FROM
  commons._events
-- you MUST specify a WHERE clause which scopes the query using :startTime and :stopTime
WHERE
  _events._event_time > :startTime AND
  _events._event_time < :stopTime
GROUP BY
  _event_time
ORDER BY
  _event_time`,
  queryParamStart: ':startTime',
  queryParamStop: ':stopTime',
  queryTimeField: '_event_time',
  queryLabelColumn: '',
};

/**
 * These are options configured for each DataSource instance
 */
export interface RocksetDataSourceOptions extends DataSourceJsonData {
  server?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface RocksetSecureJsonData {
  apiKey?: string;
}
