# Overview

The load tester receives and runs commands through its `POST /command` HTTP endpoint.
The load tester also accepts stop commands through its `GET|POST /stop` HTTP endpoint.

Typically, these commands originate from the configuration script run by `load-starter`.

This document contains the specification of test commands both in JSON format, as consumed by the `go-load-tester`, 
as well as in python configuration format, as consumed by load-starter.

# General test structure

All load tests share some basic parameters.

In configuration format the structure is:

```python
def add_vegeta_test( 
        duration:       Union[str,duration],
        test_type:      str,
        freq:           int
        per:            Union[str,duration],
        config:         Dict[str,any],
        name:           Optional[name],
        description:    Optional[str],
        url:            Optional[str],        
)
```

**NOTE:** The `url` parameter exists only in the configuration format, in the JSON format there is no URL, the
`url` is used to specify the address of the service so the service itself doesn't need it.

In JSON format the request structure is:

```JSON
{
  "name": "optional name for the test",
  "description": "optional description for the test",
  "testType": "test-type",
  "attackDuration": "5m",
  "numMessages": 100,
  "per": "1s",
  "params": { "name1":  "value1"}
}
```

There are some inconsistencies in the naming below is the table containing the corresponding conversion 
between the two formats:

| load-starter | go-load-tester | comment                                                                                                 |
|--------------|----------------|---------------------------------------------------------------------------------------------------------|
| duration     | attackDuration | the duration of the attack (using duration<br>syntax*)                                                  |
| test_type    | testType       | the test type string (see test types**)                                                                 |
| freq         | numMessages    | the number of messages per unit of time <br>(see per)                                                   |
| per          | per            | the unit of time, for the number of messages,<br>typically '1s' but it can be any duration like `5m3s`  |
| config       | params         | test dependent configuration parameters object <br>(see documentation for each test)                    |
| name         | name           | name of the test, optional <br>(used for documenting purposes)                                          |
| description  | description    | description of the test, optional<br>(used for documenting purposes)                                    |
| url          | - (nothing)    | overrides the globally set url of the load tester<br>(only used by the load-starter)                    |

## Duration parameters
Durations are specified as strings, in the configuration/python syntax they can also be specified as duration objects.

Using duration objects, in the python syntax, allows arithmetic operations, + - * / to be performed as well as
using the predefined duration constants: `Day Hour Minute Second Millisecond Microsecond Naonosecond`.

### string syntax
The string specification of duration uses the go language string syntax (summarized below).

From the go documentation:
>A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, 
>such as "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

### duration syntax

You can construct a duration using the builtin `duration` function it takes a string, in the string spec
defined above.

You can also construct a duration using arithmetic operations with existing durations for example using the
builtin duration constants: `7 * Hour + 2 * Minute` will return a duration equivalent to: 
`duration("7h2m")`.

You can mix and match durations built with the two syntaxes: `duration("7h") * 3 + Minute + 3 * Second`

# Tests

Below only the `config/params` object and `test_type/testType' field will be described since 
everything else is common and was documented above.

## Session

Here's an example of session parameters:

```json
{
  "startedRange": "1m",
  "durationRange": "2m",
  "numReleases": 3,
  "numEnvironments": 4,
  "numUsers": 5,
  "okWeight": 6,
  "exitedWeight": 7,
  "erroredWeight": 8,
  "crashedWeight": 9,
  "abnormalWeight": 10
}
```

| field           | description                                                                                                                                           |
|-----------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| startedRange    | duration range for the start of the session<br>relative to now (all generated sessions will<br>have startTime between 0 and -startRange from <br>now. |
| durationRange   | the duration of the session ( between 0 and the <br>specified duration)                                                                               |
| numReleases     | number of unique releases created                                                                                                                     |
| numEnvironments | number of unique environments created                                                                                                                 |
| numUsers        | number or unique users                                                                                                                                |
| okWeight        | relative weight of session with ok status                                                                                                             |
| exitedWeight    | relative weight of session with exited status                                                                                                         |
| erroredWeight   | relative weight of session with errored status                                                                                                        |
| crashedWeight   | relative weight of session with crashed status                                                                                                        |
| abnormalWeight  | relative weight of session with abnormal status                                                                                                       |


### Transaction

Here's an example of transaction parameters:

```json

```