# Features

## Spring Boot Configuration

Spring boot apps may be configured with an `application.yaml` file. Since yaml works quite nicely with Kubernetes objects, we support that over the properties style.

You can add configuration to your spring boot application easily using the `config` field like in [this example](https://github.com/Dante-lor/spring-boot-operator/tree/main/config/samples/configured.yaml).

!!! note "Default configurations"
    The port and context-path are defaulted in the generated application.yaml based on the `spec.port` and `spec.contextPath` properties. This is done to ensure that configuration, service settings and healthchecks can be correctly set.

## Health checks

To stop traffic heading to your spring application before it's ready, we use health checks designed around [Spring actuator](https://docs.spring.io/spring-boot/reference/actuator/enabling.html). If you haven't added spring actuator as a dependency, add this to your pom.xml file:

```xml title="pom.xml"
<dependency>
  <groupId>org.springframework.boot</groupId>
  <artifactId>spring-boot-starter-actuator</artifactId>
</dependency>
```

## Autoscaling

Your application will be equipped with a horizontal pod autoscaler which will increase and decrease the number of replicas based on cpu load.

By default, we determine the scaling behaviour from the `spec.type` field. This indicates to use which type of spring boot app you're deploying. This can be either:

| Setting   | Framework                 | Characteristics                           | CPU Target | Scale-Up       | Stabilization |
|-----------|---------------------------|-------------------------------------------|------------|----------------|---------------|
| `web`     | Spring Web                | Slower startup, higher resource usage     | 70%        | 50% / 60s      | 30s           |
| `webflux` | Spring WebFlux            | Moderate startup, more CPU efficient      | 75%        | 75% / 60s      | 20s           |
| `native`  | Spring Native (GraalVM)   | Rapid startup, highly burstable           | 65%        | 100% / 30s     | 10s           |

The default setting is `web`.

However you can override these by using the following properties:

```yaml
spec:
  autoscaler:
    minReplicas: 2 # Default value
    maxReplicas: 10 # Default value
    targetUtilization:
      cpuPercent: 70 # We may add more types in future for IO bound apps
    behavior:
      scaleUp:
        stabilizationWindowSeconds: 10
        policies:
          - type: Percent
            value: 100
            periodSeconds: 30
      scaleDown:
        stabilizationWindowSeconds: 180
```

If you want to learn more about the custom scaling behaviour, you can read more [here](https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/#configurable-scaling-behavior).

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
