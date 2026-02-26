# spring-boot-operator

The Spring Boot Operator is designed to simplify the deployment of spring boot applications in Kubernetes. When deploying applications in Kubernetes, there is so much boilerplate involved that developers just shouldn't need to care about. 

## Description

Why does a java developer have to care about whether it has read only access to the file system or what container security policy is in place. They need to care about:

* What application is running.
* How is it configured.
* How much resources does it need.
* What are it's scaling characteristics (even that is a push)

This operator simplifies the deployment of spring boot applications putting in place recommended security practices so your developers don't have to think about it.

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=dante-lor/spring-boot-operator:latest
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=dante-lor/spring-boot-operator:latest
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=dante-lor/spring-boot-operator:latest
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/dante-lor/spring-boot-operator/main/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
operator-sdk edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing

This project is very early in it's development however contributions are welcome. If you're not sure about something, feel free to post in the discussions or raise and issue and I will help you out :heart:

There is a full [Contributions Guide](./CONTRIBUTING.md) but in a nutshell:

* Use templates where they exist (for Issues and PRs)
* Try and match the code style that exists
* Follow the [KISS](https://en.wikipedia.org/wiki/KISS_principle) principle
* [Be nice to people](./CODE_OF_CONDUCT.md)

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## Updating Documentation

Our docs are built with [Material For Markdown](https://squidfunk.github.io/mkdocs-material/). To host the docs locally,
run this from inside the devcontainer:

```bash
mkdocs serve
```

From there you will be able to edit the documentation live.

## License

Copyright 2026 Daniel Taylor.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

