# Overview

This document describes the structure of a test and what one needs to do to implement a `test`.

In this document the word `test` is used to mean a load test, or a job that the go-load-tester runs in order to generate
load on a system.

go-load-tester is built on top of [Vegeta](https://github.com/tsenart/vegeta) and while reading this document does not
require understaning of how Vegeta is implemented familiarizing yourself with the principal concepts from Vegeta helps.

# General Architecture

When go-load-tester runs it starts listening for http requests containing commands for running tests, either a start
test or a stop and then it proceeds exectuing the command. A new start test command automatically stops any test that is
running at that time, that is the go-load-tester runs only one command at a time.

The load tester can run in two modes, `worker` and `master`.

A load tester that runs in `master` mode will respond to a command by dividing the requested command load to the number
of workers and generating commands for all the registered `worker` processes, It will then proceed to send the generated
commands to each worker.

A load tester that runs in `worker` mode will first send a registration request to the configured master, if there is a
master and the worker doesn't function as a standalone testrer, and then wait for requests presumably coming from the
master. Once a request has arrived the worker will proceed to execute the command by generating requests to the test
system.

For documentation regarding the exact structure of the commands see the [TestFormat document](TestFormat.md).

As the webserver receives command requests they are dispatched to the appropriate handlers.

To have a `test` recongnized by the system one needs to register it via `RegisterTestType` see below. Once a test is registered commands will be dispatched to it. As explained above depending on the mode the load tester is running in it will either break the request and forward it to the workers if in running in`master` mode  or, if running in `worker` mode it will execute the load test.  To achive this `RegisterTestType` receives 3 parameters, the test type name, a string, to be used by the dispatcher to identify the test and two functions. 

{{.RegisterTestType}}

When the load tester works in `master` mode it will use the `splitter` parameter to split the load. When working in `worker` mode the load tester will use the `tester` parameter to perform the load test.

{{.LoadSplitter}}

If the load can simply be split equaly between all `worker` processors, as is most commonly the case, just pass an `nil` splitter and the system will then use `SimpleLoadSplitter`

{{.SimpleLoadSplitter}}

{{.LoadTesterBuilder}}

The `LoadTestBuilder` returns a `LoadTester` interface which is used to perform the test. The `LoadTester` performs two functions, returns a [vegeta.Targeter](https://pkg.go.dev/github.com/tsenart/vegeta/lib#Targeter) to do the actual load test and implements a function to process the result returned by each call to the test system. 

If there is no need to process the returned results from the system under test implement an empty function for processing the result:

```go
func (tlt *myLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {
	return // nothing to do
}
```


