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
      user: "{{ env \"GITHUB_USER\" }}"
      access_token: "{{ env \"GITHUB_ACCESS_TOKEN\" }}"

  - name: customrpc
    driver: rpc
    with:
      address: "{{ env \"CUSTOM_RPC_ADDRESS\" }}"

jobs:
  - name: print task update if task is received
    on:
      tracker: gh_dastracker
      with:
        events: "issue"
    do:
      - action: customrpc/Print
        with:
          message: "Task \"{{.Update.Title}}\" has been updated and printed to the terminal."
```

This flow checks whether any issue in github is updated and sends an RPC call to Print method with the
specified message.

The configuration file uses the [go template language](https://pkg.go.dev/text/template) for placeholders.

`with` keyword specifies variables for each Action.

Helper methods:

| Method name                       | Description                                                  |
|-----------------------------------|--------------------------------------------------------------|
| env(varname string)               | returns the value of the environment variable                |
| values(m map[string]string)       | returns a list of values of the map                          |
| seq(l []string)                   | serializes the list in form of "string1,string2,string3,..." |

### Supported drivers

| Driver      | Support status                                                       |
|-------------|----------------------------------------------------------------------|
| Github      | Partially supported with issues (webhooks are not fully implemented) |
| RPC Plugins | Fully supported                                                      |

### TODO
- [ ] Predicates for triggers
- [ ] Asana support
- [ ] Jira support
- [ ] Increase test coverage

### Plugin development
The functionality of dastracker might be extended by using plugins. Each plugin is an independent process/container, implementing [Go RPC server](https://pkg.go.dev/net/rpc). Each exported method of the plugin handler must have a signature of `func(req lib.Request, res *lib.Response)`, these methods might be referred and called in the configuration.

Also, the `lib.Plugin` structure has a field, named `SetUpTrigger` with signature of `func(req SetUpTriggerReq, resp *SetUpTriggerResp) error`. This field might be filled in order to allow to hang on some triggers in the configuration file. If no method is provided, dastracker will do nothin on any trigger, hanged on the plugin.

See [example](_example/plugin/main.go) for details.

Demo: 
![](docs/demo.gif)

### Requirements
Read about requirements in [here](docs/REQUIREMENTS.md)

### Design
Read about design in [here](docs/DESIGN.md)
