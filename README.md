# go-load-tester

Load testing program, see [TestFormat.md](docs/TestFormat.md) for test parameter details.
# Usage

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
      --config string   configuration directory, .config default (default ".config")
      --log string      Log level: trace, info, warn, (error), fatal, panic (default "info")

Use "go-load-tester [command] --help" for more information about a command.

```

# Running the load tester

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
      --config string   configuration directory, .config default (default ".config")
      --log string      Log level: trace, info, warn, (error), fatal, panic (default "info")

Use "go-load-tester run [command] --help" for more information about a command.

```

## Master mode

**NOTE:** When running the load tester in master mode the server also exposes a documentation page under
the `/docs` url ( i.e. http(s)://<SERVER_ADDRESS:PORT>/docs) 

```
Usage:
  go-load-tester run master [flags]

Global Flags:
      --color                  Use color (only for console output).
      --config string          configuration directory, .config default (default ".config")
      --log string             Log level: trace, info, warn, (error), fatal, panic (default "info")
  -p, --port string            port to listen to (default "8000")
      --statsd-server string   ip:port for the statsd server
  -t, --target-url string      target URL for the attack

```

## Worker mode and standalone mode

For running the load tester in standalone mode do not provide the `master-url` parameter

```
Usage:
  go-load-tester run worker [flags]

Flags:
  -m, --master-url string   Registers worker with the specified master

Global Flags:
      --color                  Use color (only for console output).
      --config string          configuration directory, .config default (default ".config")
      --log string             Log level: trace, info, warn, (error), fatal, panic (default "info")
  -p, --port string            port to listen to (default "8000")
      --statsd-server string   ip:port for the statsd server
  -t, --target-url string      target URL for the attack

```