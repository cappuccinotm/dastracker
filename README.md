# Dastracker [![Go](https://github.com/cappuccinotm/dastracker/actions/workflows/.go.yaml/badge.svg)](https://github.com/cappuccinotm/dastracker/actions/workflows/.go.yaml) [![codecov](https://codecov.io/gh/cappuccinotm/dastracker/branch/master/graph/badge.svg?token=nLxLt9Vdyo)](https://codecov.io/gh/cappuccinotm/dastracker) [![go report card](https://goreportcard.com/badge/github.com/cappuccinotm/dastracker)](https://goreportcard.com/report/github.com/cappuccinotm/dastracker) [![Go Reference](https://pkg.go.dev/badge/github.com/cappuccinotm/dastracker.svg)](https://pkg.go.dev/github.com/cappuccinotm/dastracker)

Automating tasks management - simple and friendly, like Github Actions


### How to use
```text
Application Options:
      --dbg                     turn on debug mode [$DEBUG]

Help Options:
  -h, --help                    Show this help message

[run command options]
      -c, --config_location=    location of the configuration file
                                [$CONFIG_LOCATION]
          --update_timeout=     amount of time per processing single update
                                [$UPDATE_TIMEOUT]

    store:
          --store.type=[bolt]   type of storage [$STORE_TYPE]

    bolt:
          --store.bolt.path=    parent dir for bolt files (default: ./var)
                                [$STORE_BOLT_PATH]
          --store.bolt.timeout= bolt timeout (default: 30s)
                                [$STORE_BOLT_TIMEOUT]

    webhook:
          --webhook.base_url=   base url for webhooks [$WEBHOOK_BASE_URL]
          --webhook.addr=       local address to listen [$WEBHOOK_ADDR]
```

### DSL
Dastracker uses yaml configuration to determine which actions must happen whether some event appears, 
such as, for instance, Github issue update. The syntax is similar to Github Actions syntax. The example is above:

```yaml
trackers:
  - name: gh_dastracker
    driver: github
    with:
      owner: cappuccinotm
      name: dastracker
      user: '{{ env "GITHUB_USER" }}'
      access_token: '{{ env "GITHUB_ACCESS_TOKEN" }}'

  - name: customrpc
    driver: rpc
    with:
      address: '{{ env "CUSTOM_RPC_ADDRESS" }}'

triggers:
  - name: gh_task_updated
    in: gh_dastracker
    with:
      events: "issues"

jobs:
  - name: print task update if task is received
    on: gh_task_updated
    do:
      - if: 'string_contains(Update.Title, "[PTT]")' # means "Print To Terminal"
        do:
          - action: customrpc/Print
            detached: true
            with:
              message: |
                Task "{{.Update.Title}}" has been updated and printed to the terminal. 
                Body: 
                {{.Update.Body}}
```

This flow checks whether any issue in github is updated and sends an RPC call to Print method with the
specified message.

The configuration file uses the [go template language](https://pkg.go.dev/text/template) for placeholders 
and [expr language syntax](https://github.com/antonmedv/expr) for evaluating conditions for `if` steps.

Notice, that the condition references to update without leading dot, e.g. `Update.Title`.

`with` keyword specifies variables for each Action.

Helper methods:

| Method name                              | Description                                                  |
|------------------------------------------|--------------------------------------------------------------|
| env(varname string)                      | returns the value of the environment variable                |
| values(m map[string]string)              | returns a list of values of the map                          |
| seq(l []string)                          | serializes the list in form of "string1,string2,string3,..." |
| string_contains(s string, substr string) | returns true if string contains substring                    |

### TODO
- [X] RPC plugins support
- [ ] Github support (partially)
- [X] Predicates for triggers
- [ ] Special mappings
- [ ] Detached actions

### Terminology
- Ticket - an issue in the context of "dastracker"
- Task - an issue in the context of the end task tracker.
- Subscription - a webhook or polling, for retrieving updates from the end task tracker.

### Plugin development
The functionality of dastracker might be extended by using plugins. Each plugin is an independent process/container, 
implementing [Go RPC server](https://pkg.go.dev/net/rpc). Each exported method of the plugin handler must have a signature of `func(req lib.Request, res *lib.Response)`, 
these methods might be referred and called in the configuration.

Plugin may provide methods to subscribe and unsubscribe from events, for that, plugin should implement interface:
```go
type SubscriptionSupporter interface {
	Subscribe(req SubscribeReq, resp *SubscribeResp) error
	Unsubscribe(req UnsubscribeReq, _ *struct{}) error
}
```

See [example](_example/plugin/main.go) for details. 

Some details about JSONRPC:
- It's implemented over standard `net/rpc/jsonrpc` package, and the package itself
    receives requests over plain TCP connection. 
- Method name is prefixed with `plugin.`.
- Example of the message is follows:
    ```json
    {
      "method": "plugin.Print",
      "params": [
        {
          "ticket": {
            "id": "ticket-id",
            "task_id": "current-tracker-task-id",
            "title": "title",
            "body": "body",
            "fields": {
              "field": "value"
            }
          },
          "vars": {
            "message": "test"
          }
        }
      ],
      "id": 0
    }
    ```
