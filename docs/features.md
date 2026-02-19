# Features

## Spring Boot Configuration

Spring boot apps may be configured with an `application.yaml` file. Since yaml works quite nicely with Kubernetes objects, we support that over the properties style.

You can add configuration to your spring boot application easily using the `config` field like in [this example](https://github.com/Dante-lor/spring-boot-operator/tree/main/config/samples/configured.yaml).

!!! note "Overriding the default port"
    By default, your spring application will have the port exposed by setting `server.port` to 8080 explicitly. This is to ensure your application can be safely exposed using the service. However if you change the port by setting the server.port, the services and deployments will be exposed appropriately.


## Resource setting

By default when you provsion a [minimal spring boot application](https://github.com/Dante-lor/spring-boot-operator/tree/main/config/samples/minimal.yaml), it will be provisioned with the following resources:

```yaml
resources:
  requests:
    cpu: 1
    memory: 1Gi
  limits:
    memory: 1Gi
```

This is the default and follows the following best practices:

* Not setting CPU limits to prevent unnecessary throttling
* Setting Memory Limits to avoid OOMKilled events

This works well with container aware versions of java (11+).

You can alter this by **either** changing the `resourcePreset` or configuring the `resources` manually to specify specific values. The presets available are:

* small (1Gi memory and 1 vCPU)
* medium (2Gi memory and 2 vCPU)
* large (4Gi memory and 4vCPU)

!!! warning "Setting both won't work"
    You can't both have a preset and set the resources manually, you have to choose. If you set both, the controller will remove the preset and your custom values will be used.
