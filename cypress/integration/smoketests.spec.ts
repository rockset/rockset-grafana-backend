import { e2e } from '@grafana/e2e';
//import { RocksetDataSource } from '../../../../../../../src/components/ConfigEditor';

//import { ScenarioContext } from '@grafana/e2e/src/support';

const rsNamespace = "RocksetE2E";
const apiServer = "api.usw2a1.rockset.com";
const queryLabel = "cluster";
const testQuery = `SELECT
  TIME_BUCKET(MINUTES(5), _events._event_time) AS _event_time,
  COUNT(_events.type) AS value,
  ${queryLabel}
FROM
  commons._events
WHERE
  _events._event_time > :startTime AND
  _events._event_time < :stopTime
GROUP BY
  _event_time,
 ${queryLabel}
ORDER BY
  _event_time`;


e2e.scenario({
    describeName: 'Smoke tests',
    itName: 'Create Data Source and Rockset panel',
    addScenarioDataSource: false,
    addScenarioDashBoard: false,
    skipScenario: false,
    scenario: () => {
        // enlarge viewport so nav bar doesn't collapse and bury parts of the DOM
        e2e().viewport(2000, 1250);

        // ===== Test: setup data source =====
        e2e.flows.addDataSource({
            type: 'Rockset',
            name: rsNamespace,
            form: () => {
                e2e().get(`[data-testid="rockset api server configuration"]`).type(apiServer);
                e2e().get(`[data-testid="rockset api key configuration"]`).type(e2e.env('ROCKSET_APIKEY'));
            },
            expectedAlertMessage: 'Rockset datasource is working'
        });

        // ===== Test: Create simple Rockset panel =====
        // Setup Dashboard
        const currentDate = new Date();
        // format yyyy-mm-dd hh:mm:ss (similar to ISO)
        const toDate = currentDate.toISOString().replace('T', ' ').replace('Z', '');
        let fromDatePrime = currentDate;
        fromDatePrime.setHours(currentDate.getHours() - 6);
        const fromDate = fromDatePrime.toISOString().replace('T', ' ').replace('Z', '');

        e2e.flows.addDashboard({
            title: 'rsNamespace',
            timeRange: {
                from: fromDate,
                to: toDate
            },
        });

        // Setup Panel
        const chartData = {
            method: 'POST',
            route: '/api/ds/query',
        };
        const panelTitle = rsNamespace + " Test Panel";
        e2e.components.PageToolbar.item('Add panel').click();
        e2e.pages.AddDashboard.addNewPanel().click();

        // select datasource
        e2e.flows.selectOption({
            container: e2e.components.DataSourcePicker.container(),
            optionText: rsNamespace,
        });
        // wait on data source
        e2e().wait(2000);

        // set panel info
        e2e.components.PanelEditor.OptionsPane.fieldLabel('Panel options Title').type(`{selectall}${panelTitle}`);
        e2e().get(`[data-testid="rockset query label field"]`).clear().type(queryLabel);
        e2e().get(`[data-testid="rockset query text field"]`).clear().type(testQuery);

        // apply
        e2e().intercept(chartData.method, chartData.route).as('chartData');
        e2e.components.RefreshPicker.runButtonV2().should('be.visible').last().click();
        e2e().wait('@chartData', {timeout: 5000}).its('response.statusCode').should('eq', 200);
        e2e.components.PanelEditor.applyButton().should('be.visible').last().click();
        e2e.flows.saveDashboard();
    },
});
