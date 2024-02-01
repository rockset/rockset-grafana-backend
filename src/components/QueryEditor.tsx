import React, {ChangeEvent} from 'react';
import {InlineField, Input, TextArea} from '@grafana/ui';
import {QueryEditorProps} from '@grafana/data';
import {DataSource} from '../datasource';
import {RocksetDataSourceOptions, RocksetQuery} from '../types';

type Props = QueryEditorProps<DataSource, RocksetQuery, RocksetDataSourceOptions>;

export function QueryEditor({query, onChange, onRunQuery}: Props) {
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

    const onQueryParamLabelColumnChange = (event: ChangeEvent<HTMLInputElement>) => {
        onChange({...query, queryLabelColumn: event.target.value});
        onRunQuery();
    };

    const onQueryTextChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
        onChange({...query, queryText: event.target.value});
        onRunQuery();
    };

    const {queryText, queryParamStart, queryParamStop, queryTimeField, queryLabelColumn} = query;
    const labelWidth = 16, fieldWidth = 20;

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
                <InlineField
                    label="Label Column"
                    labelWidth={labelWidth}
                    tooltip="Name of the column containing the label to display"
                >
                    <Input
                        onChange={onQueryParamLabelColumnChange}
                        value={queryLabelColumn || ''}
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
                        style={{height: '600px'}}
                        value={queryText || ''}
                        onChange={onQueryTextChange}
                    >
                    </TextArea>
                </InlineField>
            </div>
        </>
    );
}
