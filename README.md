# Autotasker [![codecov](https://codecov.io/gh/cappuccinotm/dastracker/branch/master/graph/badge.svg?token=nLxLt9Vdyo)](https://codecov.io/gh/cappuccinotm/dastracker)

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

### Requirements
Read about requirements in [here](./docs/REQUIREMENTS.md)

### Design
Read about design in [here](./docs/DESIGN.md)
