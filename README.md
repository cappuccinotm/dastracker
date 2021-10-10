# Dastracker [![codecov](https://codecov.io/gh/cappuccinotm/dastracker/branch/master/graph/badge.svg?token=nLxLt9Vdyo)](https://codecov.io/gh/cappuccinotm/dastracker)

A tool for migrating tasks from one task tracker into another.


### How to use
    Usage:
      ___help [OPTIONS] run [run-OPTIONS]
    
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

### DSL
Dastracker uses its own language to determine which actions must happen whether some event appears, 
such as, for instance, Github issue update. The next configuration example shows the syntax:

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

The configuration file uses the go template language for placeholders.

`with` keyword specifies variables for each Action.

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
See [example](_example/plugin/main.go) for details.

Demo: 
![](docs/demo.gif)

### Requirements
Read about requirements in [here](docs/REQUIREMENTS.md)

### Design
Read about design in [here](docs/DESIGN.md)
