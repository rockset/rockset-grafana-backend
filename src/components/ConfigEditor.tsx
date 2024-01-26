import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { RocksetDataSourceOptions, RocksetSecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<RocksetDataSourceOptions> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const onServerChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      server: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  const onVIChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      vi: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  const onAPIKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        apiKey: event.target.value,
      },
    });
  };

  const onResetAPIKey = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        apiKey: '',
      },
    });
  };

  const { jsonData, secureJsonFields } = options;
  const secureJsonData = (options.secureJsonData || {}) as RocksetSecureJsonData;
  const labelWidth = 15, fieldWidth = 40

  return (
      <div>
        <div className="gf-form-group">
          <InlineField label="API Server" labelWidth={labelWidth}
                       tooltip={"for a full list of API servers, see https://docs.rockset.com/documentation/reference/rest-api"}>
            <Input
                onChange={onServerChange}
                value={jsonData.server || 'api.usw2a1.rockset.com'}
                placeholder="Rockset API server"
                width={fieldWidth}
            />
          </InlineField>
          <InlineField label="API Key" labelWidth={labelWidth}>
            <SecretInput
                isConfigured={(secureJsonFields && secureJsonFields.apiKey) as boolean}
                value={secureJsonData.apiKey || ''}
                placeholder="Rockset API key"
                width={fieldWidth}
                onReset={onResetAPIKey}
                onChange={onAPIKeyChange}
            />
          </InlineField>
        </div>
        <div className="gf-form-group">
          <InlineField label="Virtual Instance ID" labelWidth={30}
                       tooltip={"use a specific virtual instance to execute the queries"}>
            <Input
                onChange={onVIChange}
                value={jsonData.vi || ''}
                placeholder="Virtual Instance ID"
                width={60}
            />
          </InlineField>
        </div>
      </div>
  );
}
