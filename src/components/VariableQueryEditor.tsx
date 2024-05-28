import React, {ChangeEvent} from 'react';
import {InlineField, TextArea} from '@grafana/ui';
import {QueryEditorProps} from '@grafana/data';
import {DataSource} from '../datasource';
import {RocksetDataSourceOptions, RocksetQuery} from '../types';

type Props = QueryEditorProps<DataSource, RocksetQuery, RocksetDataSourceOptions>;

export function VariableQueryEditor({query, onChange, onRunQuery}: Props) {
    const onQueryTextChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
        onChange({...query, queryText: event.target.value});
        onRunQuery();
    };

    const {queryText} = query;
    const labelWidth = 16;

    const defaultQuery =
`SELECT
    TIME_BUCKET(MINUTES(5), e._event_time) AS _event_time,
    COUNT(e.type) AS count,
    e.kind AS label
FROM
    commons._events e
WHERE
    e._event_time > :startTime
    AND e._event_time < :stopTime
    AND e.kind LIKE '$kind'
GROUP BY
    e._event_time,
    e.label
ORDER BY
    e._event_time DESC`

    return (
        <div>
            <InlineField
                label="Query Text"
                labelWidth={labelWidth}
                grow={true}
                tooltip="Rockset SQL query to get the data. Must contain a WHERE clause which limits the query based on the startTime and stopTime."
            >
                <TextArea
                    style={{height: '300px'}}
                    value={queryText || defaultQuery}
                    onChange={onQueryTextChange}
                >
                </TextArea>
            </InlineField>
        </div>
    );
}
