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
`select
  e.kind
from
  commons._events e
group by
  kind
ORDER BY
  kind`

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
