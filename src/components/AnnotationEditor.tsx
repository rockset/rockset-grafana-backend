import React, {ChangeEvent, useEffect} from 'react';
import {InlineField, Input, TextArea} from '@grafana/ui';
import {QueryEditorProps} from '@grafana/data';
import {DataSource} from '../datasource';
import {RocksetDataSourceOptions, RocksetQuery} from '../types';

type Props = QueryEditorProps<DataSource, RocksetQuery, RocksetDataSourceOptions>;

export function AnnotationEditor({query, onChange, onRunQuery}: Props) {
    useEffect(() => {
        onChange({...query, queryParamStart: ':startTime', queryParamStop: ':stopTime'});
    }, [onChange]); // eslint-disable-line react-hooks/exhaustive-deps

    const onQueryParamStartChange = (event: ChangeEvent<HTMLInputElement>) => {
        onChange({...query, queryParamStart: event.target.value});
        onRunQuery();
    };

    const onQueryParamStopChange = (event: ChangeEvent<HTMLInputElement>) => {
        onChange({...query, queryParamStop: event.target.value});
        onRunQuery();
    };

    const onQueryParamTimeFieldChange = (event: ChangeEvent<HTMLInputElement>) => {
        onChange({...query, queryTimeField: event.target.value});
        onRunQuery();
    };

    const onQueryTextChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
        onChange({...query, queryText: event.target.value});
        onRunQuery();
    };

    const {queryText, queryParamStart, queryParamStop,queryTimeField} = query;
    const labelWidth = 16, fieldWidth = 20;

    const defaultQuery = `SELECT
    e._event_time,
    CASE
        WHEN e.message IS NOT NULL THEN e.message
        ELSE 'no text found'
    END AS text,
    FORMAT('type={},kind={}', e.type, e.kind) AS tags
FROM
    commons._events e
WHERE
  e._event_time > :startTime AND
  e._event_time < :stopTime
ORDER BY
    time DESC`

    return (
        <>
            <div className="gf-form">
                <InlineField
                    label="Time Column"
                    labelWidth={labelWidth}
                    tooltip="Name of the column containing the time series"
                >
                    <Input
                        onChange={onQueryParamTimeFieldChange}
                        value={queryTimeField || '_event_time'}
                        width={fieldWidth}
                    />
                </InlineField>
                <InlineField
                    label="Start Time"
                    labelWidth={labelWidth}
                    tooltip="Name of the query parameter for the start time"
                >
                    <Input
                        onChange={onQueryParamStartChange}
                        value={queryParamStart || ':startTime'}
                        width={fieldWidth}
                    />
                </InlineField>
                <InlineField
                    label="Stop Time"
                    labelWidth={labelWidth}
                    tooltip="Name of the query parameter for the start time"
                >
                    <Input
                        onChange={onQueryParamStopChange}
                        value={queryParamStop || ':stopTime'}
                        width={fieldWidth}
                    />
                </InlineField>
            </div>
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
        </>
    );
}
