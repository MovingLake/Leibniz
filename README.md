# Leibniz
## Golang framework to bootstrap API-to-API integration projects.

[![Go Reference](https://pkg.go.dev/badge/golang.org/x/example.svg)](https://pkg.go.dev/github.com/movinglake/leibniz)

Leibniz is a batteries-included framework to kickstart any integration project. It includes scheduling recurring tasks for pulling data as well as an HTTP server to support webhooks.

Named after [Gottfried Wilhelm Leibniz](https://en.wikipedia.org/wiki/Gottfried_Wilhelm_Leibniz) one of the discoverers of calculus including *integrals*.

## Clone the project

```
$ git clone https://github.com/MovingLake/Leibniz.git
$ cd Leibniz
```

## [Sample code](main.go)

```
$ go build .
$ ./leibniz
```
Launches the example which contains an [example taskrunner](taskrunner/example.go) as well as an [example  httpendpoint](httpendpoints/example.go). It also adds a single recurring task that generates tasks every minute. The HTTP server is launched to serve on port 8080.

## Leibniz [configuration](launch_config.json)

Leibniz framework needs a json configuration file to set all the pertinent variables. In addition it uses one single environment variable called `LEIBNIZ_LAUNCH_CONFIG_FILE` to designate where the configuration file lives. It uses `launch_config.json` by default.

## [Logging](lib/log.go)

Leibniz uses a lightweight wrapper around the `log` library to write logs. The log level can be configured through the `launch_config.json` file, but by default it emits all logs. We recommend limiting to `warn` or `error` levels for production environments.

## Demo repo [Github to Slack](github.com/movinglake/leibniz-demo)

Checkout also this repo for more examples on how to use Leibniz.

## Testing

Leibniz has 100% unit test coverage. It does not use stubs, mock or fakes for the database, but needs a local postgres DB to run all unit tests.