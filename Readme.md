# Woodpecker CI Configuration Service example

This repository provides a very simplistic example of how to set up an external configuration service for Woodpecker CI
The external service gets a HTTP POST request with information about the repo, current build, and the configs that would normally be used. 
It can then decide to acnowledge the current configs (By returning HTTP 204), or overriding the configurations and returning new ones in the response

Usecases for this system are: 
- Centralized configuration for multiple repositories at once
- Preprocessing steps in the pipeline like templating, macros or conversion from different pipeline formats to woodpeckers format

This service is written in go, run using `go run .`. Then configure your woodpecker instance to point to it and configure the same secret. See [Woodpeckers documentation here](https://woodpecker-ci.org/docs/administration/external-configuration-api)

eg: 

```shell
# Server
# ...
WOODPECKER_CONFIG_SERVICE_ENDPOINT=http://<service>:8000/ciconfig
WOODPECKER_CONFIG_SERVICE_SECRET=mysecretsigningkey

```

