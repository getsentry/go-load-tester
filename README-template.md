# go-load-tester

Load testing program, see [TestFormat.md](docs/TestFormat.md) for test parameter details.
# Usage

The load tester can run in a few modes:
* as a master process controlling worker load testers
* as a worker load tester
* as a standalone load tester ( this is achieved by running it in worker mode without providing a master url)

Global usage

```
{{.RootUsage}}
```

# Running the load tester

```
{{.RunUsage}}
```

## Master mode

**NOTE:** When running the load tester in master mode the server also exposes a documentation page under
the `/docs` url ( i.e. http(s)://<SERVER_ADDRESS:PORT>/docs) 

```
{{.MasterUsage}}
```

## Worker mode and standalone mode

For running the load tester in standalone mode do not provide the `master-url` parameter

```
{{.WorkerUsage}}
```