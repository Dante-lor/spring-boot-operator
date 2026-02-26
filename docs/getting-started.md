# Getting started

To get started with the Spring Boot Operator, you will need to install the Operator to your cluster. For production settings, we recommend using the [Operator Lifecycle Manager](./installation/olm.md) to install the operator.

## Prerequisites

* Your kubernetes cluster is at v1.30+
* Your application uses java 11+
* Your application uses spring boot 2+

## Quickstart

To install the latest version of the operator, ensure you have [Cert Manager](https://cert-manager.io/docs/) installed in your cluster. Then you can install using `kubectl`:

```bash
kubectl apply -f https://github.com/Dante-lor/spring-boot-operator/releases/download/v0.1.2/install.yaml
```

## Next steps

Assuming you meet the prerequisites and have installed the opartor, deploying a vanilla spring application is as simple as:

```bash
DOCKER_IMAGE=example:latest

kubectl apply -f - <<EOF
apiVersion: spring.dante-lor.github.io/v1alpha1
kind: SpringBootApplication
metadata:
  name: my-app
spec:
  image: $DOCKER_IMAGE
EOF
```

If you want to dive deeper into what features and configuration options are available, check out our [Feature page](./features.md).
