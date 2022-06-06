# Grafana Rockset Data Source Backend Plugin

[![CircleCI](https://circleci.com/gh/rockset/rockset-grafana/tree/master.svg?style=svg)](https://circleci.com/gh/grafana/simple-datasource-backend/tree/master)

The Rockset plugin lets you write queries against your Rockset collections and visualize the
results as Grafana graphs. 

https://docs.rockset.com/grafana/

The query has two required query parameters, named `:startTime` and `:endTime` by default, which must be used
in a `WHERE` clause to scope the query to the selected time period in Grafana (or you will end up querying
your entire collection).

A sample query to graph Rockset events by 5 minute intervals

```
SELECT
    TIME_BUCKET(MINUTES(5), _events._event_time) AS _event_time,
    COUNT(_events.type) AS value
FROM
    commons._events
WHERE
    _events._event_time > :startTime AND
    _events._event_time < :stopTime
GROUP BY
    _event_time
ORDER BY
    _event_time
```

You can use one column of the result to label the data, e.g. in the below query the `type` is the label column

```
SELECT
    TIME_BUCKET(MINUTES(5), _events._event_time) AS _event_time,
    _events.type,
    COUNT(_events.type) AS value
FROM
    commons._events
WHERE
    _events._event_time > :startTime AND
    _events._event_time < :stopTime
GROUP BY
    _event_time,
    type
ORDER BY
    _event_time
```

## Development

The Rockset data source backend plugin consists of both frontend and backend components.

### Frontend

1. Install dependencies
```BASH
yarn install
```

2. Build plugin in development mode or run in watch mode
```BASH
yarn dev
```
or
```BASH
yarn watch
```
3. Build plugin in production mode
```BASH
yarn build
```

### Backend

1. Setup `mage`

```BASH

```

2. Build backend plugin binaries for Linux, Windows and Darwin:
```BASH
mage -v
```

3. List all available Mage targets for additional commands:
```BASH
mage -l
```

## Testing

Run grafana in a docker container

```
docker run -d \
    -p 3000:3000 \
    -v "$(pwd)/..:/var/lib/grafana/plugins" \
    -e "GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=rockset" \
    --name=grafana \
    grafana/grafana:7.0.3
```

Since Grafana only loads plugins on start-up, you need to restart the container whenever you add or remove a plugin.

```
docker restart grafana
```

## Learn more

- [Build a data source backend plugin tutorial](https://grafana.com/tutorials/build-a-data-source-backend-plugin)
- [Grafana documentation](https://grafana.com/docs/)
- [Grafana Tutorials](https://grafana.com/tutorials/) - Grafana Tutorials are step-by-step guides that help you make the most of Grafana
- [Grafana UI Library](https://developers.grafana.com/ui) - UI components to help you build interfaces using Grafana Design System
- [Grafana plugin SDK for Go](https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/)
