# Environment

## Pre-requisites

* gcc
* golang 1.11 or higher (to support go modules)

# Environment variables
* You may need to add `{hindsight_dir}/agent` to your `$GOPATH`
* Hindsight requires the following variables for using cgo:
```
export CGO_LDFLAGS_ALLOW=".*"
```
* (Unsure) maybe increase gomaxprocs:
```
export GOMAXPROCS=10
```

# Configuring 

Configuration is not required to run local or integration tests, but is required for a full deployment.  See [configuration.md](configuration.md).
