import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { RocksetDataSourceOptions, RocksetSecureJsonData } from '../types';

const { SecretFormField, FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<RocksetDataSourceOptions> {};

interface State {};

const apiServerDataTestId = "rockset api server configuration";
const apiKeyDataTestId = "rockset api key configuration";

export class ConfigEditor extends PureComponent<Props, State> {
  setServer = (server: string) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      server: server,
    };
    onOptionsChange({ ...options, jsonData });
  };

  onServerChange = (event: ChangeEvent<HTMLInputElement>) => {
    this.setServer(event.target.value);
  };

  // Secure field (only sent to the backend)
  onAPIKeyChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        apiKey: event.target.value,
      },
    });
  };

  onResetAPIKey = () => {
    const { onOptionsChange, options } = this.props;
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

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as RocksetSecureJsonData;

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <FormField
            label="API server"
            labelWidth={6}
            inputWidth={20}
            onChange={this.onServerChange}
            value={jsonData.server || ''}
            placeholder="api server (https://rockset.com/docs/rest-api/)"
            data-testid={apiServerDataTestId}
          />
        </div>

        <div className="gf-form-inline">
          <div className="gf-form">
            <SecretFormField
              isConfigured={(secureJsonFields && secureJsonFields.apiKey) as boolean}
              value={secureJsonData.apiKey || ''}
              label="API Key"
              placeholder="Rockset API Key"
              labelWidth={6}
              inputWidth={64}
              onReset={this.onResetAPIKey}
              onChange={this.onAPIKeyChange}
              data-testid={apiKeyDataTestId}
            />
          </div>
        </div>
      </div>
    );
  }
};
