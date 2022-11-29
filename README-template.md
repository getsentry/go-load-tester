# go-load-tester

`go-load-tester` is a distributed load testing framework. It is based on [Vegeta](https://github.com/tsenart/vegeta) and can run in a distributed mode (something that Vegeta cannot do at the moment), borrowing the approach from [Locust](https://github.com/locustio/locust/).

**NOTE:** this is a work in progress.

Known limitations:

* The framework is currently Sentry-oriented. The data generators that are currently supported are described [here](docs/TestFormat.md#tests).
* Worker registration/keep-alive behaviour is not very robust
* Sentry error generation

## Usage

For supported load generators and parameter details -- see [here](docs/TestFormat.md).

[More information about the general architecture and writing tests.](docs/Writing-tests.md)


The load tester can run in a few modes:
* as a master process controlling worker load testers
* as a worker load tester
* as a standalone load tester ( this is achieved by running it in worker mode without providing a master url)

Global usage

```
{{.RootUsage}}
```

## Running Load Tester

```
{{.RunUsage}}
```

### Master Mode

**NOTE:** When running the load tester in master mode the server also exposes a documentation page under
the `/docs` url ( i.e. http(s)://<SERVER_ADDRESS:PORT>/docs)

```
{{.MasterUsage}}
```

### Worker Mode and Standalone Mode

For running the load tester in standalone mode do not provide the `master-url` parameter

```
{{.WorkerUsage}}
```
