# Backend component

The backend component contains everything server-side which connects to the database, message queue, and Minecraft server. Once running, it will start the HTTP API server which is consumed by the frontend client application, configure RCON connection to your Minecraft server, and spawn worker process that waits and continuous handles incoming whitelist applications and dispatching tasks to Ops. It is implemented with reliability and security in mind which supports automatic channel recovery should connection to message broker got closed unexpectedly and retry patterns for operations in case of failure

## Local setup

`config.yaml` needs to be placed in the root directory of this component and configured correctly according to your setup.

`go run cmd/main.go` will start the backend component.
Watch for logs output to see if everything is set up correctly.

## config.yaml

`config.yaml` file serves the centralized place for server-side configuration. See `config_sample.yaml` for  detailed explanation of each option. The config entries in this file represent the same set of entries as in the backend helm chart's `values.yaml` which is used for Kubernetes deployment.
