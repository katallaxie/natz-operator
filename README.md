# Kubernetes Operator for NATS Accounting

[![Release](https://github.com/katallaxie/natz-operator/actions/workflows/release.yml/badge.svg)](https://github.com/katallaxie/natz-operator/actions/workflows/release.yml)
[![Taylor Swift](https://img.shields.io/badge/secured%20by-taylor%20swift-brightgreen.svg)](https://twitter.com/SwiftOnSecurity)
[![Volkswagen](https://auchenberg.github.io/volkswagen/volkswargen_ci.svg?v=1)](https://github.com/auchenberg/volkswagen)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Kubernetes operator for [NATS](https://nats.io/) accounting.

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/katallaxie/natz-operator?quickstart=1)

## Installation

[Helm](https://helm.sh/) can be used to install the `natz-operator` to your Kubernetes cluster.

```shell
helm repo add natz-operator https://katallaxie.github.io/natz-operator/helm/charts
helm repo update
helm search repo natz-operator
```

## Usage

There are three custom account resources that can be used to configure the operator.

- `NatsKey`
- `NatsOperator`
- `NatsAccount`
- `NatsUser`
- `NatsConfiguration`
- `NatsGateway`
- `NatsConfig`
- `NatsActivation`

These can be configured with `NatsKey` to provide a private key and additional signing keys for the operator and accounts.

Creating the operator for the [NATS](https://nats.io/) accounting.

```yaml
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsKey
metadata:
  name: natsoperator-sample-private-key
spec:
  type: Operator
---
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsKey
metadata:
  name: natsoperator-demo-signing-key
spec:
  type: Operator
---
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsOperator
metadata:
  name: natsoperator-sample
spec:
  privateKey:
    name: natsoperator-sample-private-key
  signingKeys:
    - name: natsoperator-demo-signing-key
```

Creating the system account for the operator.

```yaml
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsKey
metadata:
  name: natsoperator-system-private-key
spec:
  type: Account
---
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsKey
metadata:
  name: natsoperator-system-signing-key
spec:
  type: Account
---
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsAccount
metadata:
  name: natsoperator-system
spec:
  signerKeyRef:
    name: natsoperator-sample-private-key
  privateKey:
    name: natsoperator-system-private-key
  signingKeys:
    - name: natsoperator-system-signing-key
  exports:
    - name: account-monitoring-services
      subject: $SYS.REQ.ACCOUNT.*.*
      type: 2
      response_type: Stream
      account_token_position: 4
      description: "Request account specific monitoring services for: SUBSZ, CONNZ, LEAFZ, JSZ and INFO"
      info_url: "https://docs.nats.io/nats-server/configuration/sys_accounts"
    - name: account-monitoring-streams
      subject: $SYS.ACCOUNT.*.>"
      type: 1
      account_token_position: 3
      description: "Account specific monitoring stream"
      info_url: "https://docs.nats.io/nats-server/configuration/sys_accounts"
  limits:
    exports: -1
    imports: -1
    subs: -1
    payload: -1
    data: -1
    conn: -1
    wildcards: true
    disallow_bearer: true

```

Creating a user account.

```yaml
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsUser
metadata:
  name: knative-eventing-user
spec:
  accountRef:
    namespace: default
    name: knative-eventing-account
  limits:
    payload: -1
    subs: -1
    data: -1
```

## NATS Configuration

The NATS configuration can be created using the following configuration.

```yaml
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsConfig
metadata:
  name: nats-default-config
spec:
  operatorRef:
    name: natsoperator-sample
  systemAccountRef:
    name: natsoperator-system
  config:
    host: 0.0.0.0
    port: 4222
  gateways:
    - name: harry
      namespace: default
```

This creates a new `Secret` with the NATS configuration. 
This configuration can be merged with the NATS operator configuration.

Gateways are dynamically configured using the `NatsGateway` resources in the `gateways` property.

There are dynamic 

## Gateways

NATS gateways can be created using the following configuration.

:warning: do not store passwords in the YAMl configuration files.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gateway-north-secret
data:
  username: demo
  password: NjJlYjE2NWMwNzBhNDFkNWMxYjU4ZDlkM2Q3MjVjYTE=
---
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsGateway
metadata:
  name: harry
spec:
  url: nats://nats.north:4222
  username:
    secretKeyRef:
      key: username
      name: gateway-north-secret
  password:
    secretKeyRef:
      key: password
      name: gateway-north-secret
```

## NATS Operator

In order to create a configuration for the NATS operator, you can use the following configuration.

```yaml
apiVersion: natz.katallaxie.com/v1alpha1
kind: NatsConfig
metadata:
  name: nats-default-config
spec:
  operatorRef:
    name: natsoperator-sample
  systemAccountRef:
    name: natsoperator-system
```

The operator can be integrated with the NATS operator.

```yaml
natsBox:
  enabled: true

config:
  jetstream:
    enabled: true
    fileStore:
      pvc:
        size: 10Gi

statefulSet:
  patch:
    - op: remove
      path: /spec/template/spec/volumes/0
    - op: add
      path: /spec/template/spec/volumes/-
      value:
        name: config
        secret:
          defaultMode: 420
          secretName: nats-default-config
```

## Development

You can use [kind](https://kind.sigs.k8s.io/) to test the operator.

```shell
kind create cluster
```

The operator can be built and tested using the following commands.

```shell
make generate
make install
```

Then you can start the operator using the following command.

```shell
make up
```

## License

[Apache 2.0](/LICENSE)
