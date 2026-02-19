# Installing the Spring Boot Operator via OLM

The [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/) is the recommended way to install and manage the Spring Boot Operator in a cluster. OLM handles installation, upgrades, and dependency management automatically.

---

## Prerequisites

Before you begin, ensure the following are available in your environment:

- A running Kubernetes cluster (v1.30+)
- `kubectl` configured to point to your cluster
- OLM installed in the cluster (see below if not already installed)
- Sufficient permissions to create cluster-scoped resources (ClusterRole, CatalogSource, etc.)

---

## Step 1 — Install OLM (if not already present)

If your cluster does not have OLM installed, you can install it using the [Operator SDK CLI](https://sdk.operatorframework.io/docs/installation/):

```bash
operator-sdk olm install
```

To verify OLM is running:

```bash
kubectl get pods -n olm
```

All pods in the `olm` namespace should be in the `Running` state before proceeding.

---

## Step 2 — Add the Catalog Source

The catalog source tells OLM where to find the Spring Boot Operator bundle. Apply the following manifest to register it with your cluster:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: spring-boot-operator-catalog
  namespace: olm
spec:
  sourceType: grpc
  image: ghcr.io/dante-lor/spring-boot-operator-catalog:latest
  displayName: Spring Boot Operator
  publisher: dante-lor
  updateStrategy:
    registryPoll:
      interval: 10m
```

Apply it with:

```bash
kubectl apply -f catalogsource.yaml
```

Verify the catalog source is ready:

```bash
kubectl get catalogsource -n olm spring-boot-operator-catalog
```

The `STATUS` column should show `READY`.

---

## Step 3 — Create a Namespace and OperatorGroup

OLM requires an `OperatorGroup` to define the install scope of the operator. The example below installs the operator into a dedicated namespace and configures it to watch all namespaces. Adjust `targetNamespaces` to restrict the watch scope if needed.

```bash
kubectl create namespace spring-boot-operator-system
```

```yaml
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: spring-boot-operator-group
  namespace: spring-boot-operator-system
spec:
  # Omitting targetNamespaces causes the operator to watch all namespaces.
  # To restrict to specific namespaces, add them as a list:
  # targetNamespaces:
  #   - my-app-namespace
```

```bash
kubectl apply -f operatorgroup.yaml
```

---

## Step 4 — Create a Subscription

The `Subscription` resource instructs OLM to install the operator and keep it up to date:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: spring-boot-operator
  namespace: spring-boot-operator-system
spec:
  channel: alpha
  name: spring-boot-operator
  source: spring-boot-operator-catalog
  sourceNamespace: olm
  # Pin to a specific version by setting startingCSV, e.g.:
  # startingCSV: spring-boot-operator.v0.0.1
```

```bash
kubectl apply -f subscription.yaml
```

---

## Step 5 — Verify the Installation

OLM will now install the operator. Track the progress by checking the `ClusterServiceVersion` (CSV):

```bash
kubectl get csv -n spring-boot-operator-system
```

Wait until the `PHASE` column shows `Succeeded`:

```
NAME                            DISPLAY                 VERSION   REPLACES   PHASE
spring-boot-operator.v0.0.1    Spring Boot Operator    0.0.1                Succeeded
```

You can also verify the operator pod is running:

```bash
kubectl get pods -n spring-boot-operator-system
```

---

## Upgrading

When a new version of the catalog image is published, OLM will detect it automatically based on the `registryPoll` interval defined in the `CatalogSource` (default: 10 minutes). If you are subscribed to a channel, the upgrade will be applied automatically.

To trigger an immediate check, you can delete and recreate the catalog source pod:

```bash
kubectl delete pod -n olm -l olm.catalogSource=spring-boot-operator-catalog
```

To pin to a specific version and manage upgrades manually, set `installPlanApproval: Manual` in your `Subscription`:

```yaml
spec:
  installPlanApproval: Manual
```

With manual approval, you must approve each `InstallPlan` as it is created:

```bash
# List pending install plans
kubectl get installplan -n spring-boot-operator-system

# Approve a specific install plan
kubectl patch installplan <install-plan-name> \
  -n spring-boot-operator-system \
  --type merge \
  --patch '{"spec":{"approved":true}}'
```

---

## Uninstalling

To remove the operator and all associated resources:

```bash
# Delete the subscription to stop future upgrades
kubectl delete subscription spring-boot-operator -n spring-boot-operator-system

# Delete the CSV to remove the operator itself
kubectl delete csv spring-boot-operator.v0.0.1 -n spring-boot-operator-system

# Remove the OperatorGroup and namespace
kubectl delete operatorgroup spring-boot-operator-group -n spring-boot-operator-system
kubectl delete namespace spring-boot-operator-system

# Remove the catalog source
kubectl delete catalogsource spring-boot-operator-catalog -n olm
```

> **Note:** Deleting the operator does not automatically remove Custom Resource Definitions (CRDs) or any custom resources you have created. Remove these manually if they are no longer needed to avoid orphaned resources in your cluster.

---

## Troubleshooting

**CatalogSource not becoming READY**
Check the catalog pod logs in the `olm` namespace:
```bash
kubectl get pods -n olm -l olm.catalogSource=spring-boot-operator-catalog
kubectl logs -n olm <catalog-pod-name>
```
Ensure the catalog image is publicly accessible from your cluster or that image pull secrets are configured.

**CSV stuck in `Installing` or `Failed` phase**
Inspect the CSV events and status:
```bash
kubectl describe csv spring-boot-operator.v0.0.1 -n spring-boot-operator-system
```

**No packages found in subscription**
Confirm the `source` and `sourceNamespace` in your `Subscription` match the `name` and `namespace` of your `CatalogSource` exactly.

---

## Additional Resources

- [OLM Documentation](https://olm.operatorframework.io/docs/)
- [Operator SDK OLM Integration Guide](https://sdk.operatorframework.io/docs/olm-integration/)
- [Spring Boot Operator Releases](https://github.com/dante-lor/spring-boot-operator/releases)