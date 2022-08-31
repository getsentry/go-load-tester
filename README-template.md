# go-load-tester

Load testing program, see _Documents for test parameter details.
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

```
{{.MasterUsage}}
```

## Worker mode and standalone mode

For running the load tester in standalone mode do not provide the `master-url` parameter

```
{{.WorkerUsage}}
```