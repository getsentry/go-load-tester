# go-load-tester

`go-load-tester` is a distributed load testing framework. It is based on [Vegeta](https://github.com/tsenart/vegeta) and can run in a distributed mode (something that Vegeta cannot do at the moment), borrowing the approach from [Locust](https://github.com/locustio/locust/).

**NOTE:** this is a work in progress.

Known limitations:

* The framework is currently Sentry-oriented. The data generators that are currently supported are described [here](docs/TestFormat.md#tests).
* Worker registration/keep-alive behaviour is not very robust

## Usage

For supported load generators and parameter details -- see [here](docs/TestFormat.md).

[More information about the general architecture and writing tests.](docs/Writing-tests.md)


The load tester can run in a few modes:
* as a master process controlling worker load testers
* as a worker load tester
* as a standalone load tester ( this is achieved by running it in worker mode without providing a master url)

Global usage

```
Usage:
  go-load-tester [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  run         Runs the load tester
  update-docs Extract docs from source code into static files.

Flags:
      --color           Use color (only for console output).
      --config string   configuration directory (default ".config")
      --log string      Log level: trace, info, warn, (error), fatal, panic (default "info")

Use "go-load-tester [command] --help" for more information about a command.

```

## Running Load Tester

```
Usage:
  go-load-tester run [command]

Available Commands:
  master      Run load tester in master mode.
  worker      Run a worker, that waits for commands from a server

Flags:
  -p, --port string            port to listen to (default "8000")
      --statsd-server string   ip:port for the statsd server
  -t, --target-url string      target URL for the attack

Global Flags:
      --color           Use color (only for console output).
      --config string   configuration directory (default ".config")
      --log string      Log level: trace, info, warn, (error), fatal, panic (default "info")

Use "go-load-tester run [command] --help" for more information about a command.

```

### Master Mode

**NOTE:** When running the load tester in master mode the server also exposes a documentation page under
the `/docs` url ( i.e. http(s)://<SERVER_ADDRESS:PORT>/docs)

```
Usage:
  go-load-tester run master [flags]

Global Flags:
      --color                  Use color (only for console output).
      --config string          configuration directory (default ".config")
      --log string             Log level: trace, info, warn, (error), fatal, panic (default "info")
  -p, --port string            port to listen to (default "8000")
      --statsd-server string   ip:port for the statsd server
  -t, --target-url string      target URL for the attack

```

### Worker Mode and Standalone Mode

For running the load tester in standalone mode do not provide the `master-url` parameter

```
Usage:
  go-load-tester run worker [flags]

Flags:
  -m, --master-url string   Registers worker with the specified master

Global Flags:
      --color                  Use color (only for console output).
      --config string          configuration directory (default ".config")
      --log string             Log level: trace, info, warn, (error), fatal, panic (default "info")
  -p, --port string            port to listen to (default "8000")
      --statsd-server string   ip:port for the statsd server
  -t, --target-url string      target URL for the attack

```
