# Getting started

To get started with the Spring Boot Operator, you will need to install the Operator to your cluster. For production settings, we recommend using the [Operator Lifecycle Manager](./installation/olm.md) to install the operator.

## Quickstart

To install the latest version of the operator, ensure you have [Cert Manager](https://cert-manager.io/docs/) installed in your cluster. Then you can install using `kubectl`:

```bash
kubectl apply -f https://github.com/Dante-lor/spring-boot-operator/releases/download/v0.0.1/install.yaml
```
