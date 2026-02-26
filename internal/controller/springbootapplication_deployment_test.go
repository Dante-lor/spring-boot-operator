package controller

import (
	"context"

	"github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	springv1alpha1 "github.com/dante-lor/spring-boot-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	res "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Deployment Controller", func() {
	const resourceName = "test-deploy"
	const namespace = "default"

	var (
		ctx                  context.Context
		typeNamespacedName   types.NamespacedName
		controllerReconciler *SpringBootApplicationReconciler
		app                  *springv1alpha1.SpringBootApplication
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		app = &springv1alpha1.SpringBootApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: springv1alpha1.SpringBootApplicationSpec{
				Type:           springv1alpha1.SpringWeb,
				Image:          "test",
				ResourcePreset: ptr.To(v1alpha1.Small),
				Autoscaler: springv1alpha1.AutoscalingConfig{
					MinReplicas: 1,
					MaxReplicas: 5,
				},
			},
		}

		controllerReconciler = &SpringBootApplicationReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		By("creating the SpringBootApplication resource")
		Expect(k8sClient.Create(ctx, app)).To(Succeed())
	})

	AfterEach(func() {
		By("deleting the SpringBootApplication resource")
		Expect(k8sClient.Delete(ctx, app)).To(Succeed())
	})

	CheckExpectedPresetBehaviour := func(preset *springv1alpha1.ResourcePreset, expectedCPUStr string, expectedMemoryStr string) {
		app.Spec.ResourcePreset = preset

		Expect(k8sClient.Update(ctx, app)).To(Succeed())

		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: typeNamespacedName,
		})
		Expect(err).NotTo(HaveOccurred())

		dep := &appsv1.Deployment{}
		err = k8sClient.Get(ctx, typeNamespacedName, dep)
		Expect(err).NotTo(HaveOccurred())

		expectedCPU := res.MustParse(expectedCPUStr)
		expectedMem := res.MustParse(expectedMemoryStr)

		deployResources := dep.Spec.Template.Spec.Containers[0].Resources

		Expect(*deployResources.Requests.Cpu()).To(BeIdenticalTo(expectedCPU))
		Expect(*deployResources.Requests.Memory()).To(BeIdenticalTo(expectedMem))
		Expect(*deployResources.Limits.Memory()).To(BeIdenticalTo(expectedMem))

		// Checking the CPU would yield an empty resource value. Therefore checking the length to
		// Ensure only one value is set (the memory as previously tested)
		Expect(len(deployResources.Limits)).To(BeIdenticalTo(1))
	}

	It("sets resource requests and memory limits for small preset", func() {
		CheckExpectedPresetBehaviour(ptr.To(springv1alpha1.Small), "1", "1Gi")
	})

	It("sets resource requests and memory limits for medium preset", func() {
		CheckExpectedPresetBehaviour(ptr.To(springv1alpha1.Medium), "2", "2Gi")
	})

	It("sets resource requests and memory limits for large preset", func() {
		CheckExpectedPresetBehaviour(ptr.To(springv1alpha1.Large), "4", "4Gi")
	})

	It("uses custom resources if defined and preset is nil", func() {
		app.Spec.Resources = &springv1alpha1.ResourceDefinition{
			CPU:    "2",
			Memory: "8Gi",
		}

		CheckExpectedPresetBehaviour(nil, "2", "8Gi")
	})

	Describe("with small preset", func() {

		BeforeEach(func() {
			app.Spec.ResourcePreset = ptr.To(springv1alpha1.Small)
			Expect(k8sClient.Update(ctx, app)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())
		})

		It("mounts the config at /config", func() {
			deploy := &appsv1.Deployment{}

			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

			volumeName := "config"
			// Check volume is added
			volumes := deploy.Spec.Template.Spec.Volumes
			Expect(volumes).To(HaveLen(1))
			vol := volumes[0]
			Expect(vol.Name).To(Equal(volumeName))
			Expect(vol.ConfigMap.LocalObjectReference.Name).To(Equal(app.Name))

			// Check it's mounted at /config
			mounts := deploy.Spec.Template.Spec.Containers[0].VolumeMounts
			Expect(mounts).To(HaveLen(1))
			configMount := mounts[0]
			Expect(configMount.Name).To(Equal(volumeName))
			Expect(configMount.MountPath).To(Equal("/config"))
		})

		It("adds environment variable to tell where additional properties are located", func() {
			deploy := &appsv1.Deployment{}

			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())
			env := deploy.Spec.Template.Spec.Containers[0].Env

			Expect(env).To(HaveLen(2))
			configEnvVar := env[0]

			Expect(configEnvVar.Name).To(Equal("SPRING_CONFIG_ADDITIONAL_LOCATION"))
			Expect(configEnvVar.Value).To(Equal("/config"))
		})

		It("uses 70 percent of available memory for the java heap", func() {
			deploy := &appsv1.Deployment{}

			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())
			env := deploy.Spec.Template.Spec.Containers[0].Env

			Expect(env).To(HaveLen(2))
			configEnvVar := env[1]

			Expect(configEnvVar.Name).To(Equal("JAVA_TOOL_OPTIONS"))
			Expect(configEnvVar.Value).To(Equal("-XX:MaxRAMPercentage=70"))
		})
	})

	Describe("when the port is overridden", func() {
		BeforeEach(func() {
			app.Spec.ResourcePreset = ptr.To(springv1alpha1.Small)

			app.Spec.Port = 3333

			Expect(k8sClient.Update(ctx, app)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())
		})

		It("exposes the port in the deployment", func() {
			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

			ports := deploy.Spec.Template.Spec.Containers[0].Ports

			Expect(ports).To(HaveLen(1))
			exposed := ports[0]

			Expect(exposed.ContainerPort).To(BeEquivalentTo(3333))
		})
	})

	Describe("Readiness probes", func() {
		It("should use context path and port when context path has no trailing slash", func()  {
			app.Spec.ContextPath = "/mypath"
			app.Spec.Port = 8000

			Expect(k8sClient.Update(ctx, app)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

			container := deploy.Spec.Template.Spec.Containers[0]

			Expect(*container.ReadinessProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/mypath/actuator/health/readiness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}))

			Expect(*container.LivenessProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/mypath/actuator/health/liveness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}))

			Expect(*container.StartupProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/mypath/actuator/health/liveness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 30, // Wait 5 minutes before declaring failure
			}))
		})

		It("should use context path and port when context path has trailing slash", func()  {
			app.Spec.ContextPath = "/mypath/"
			app.Spec.Port = 8000

			Expect(k8sClient.Update(ctx, app)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

			container := deploy.Spec.Template.Spec.Containers[0]

			Expect(*container.ReadinessProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/mypath/actuator/health/readiness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}))

			Expect(*container.LivenessProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/mypath/actuator/health/liveness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}))

			Expect(*container.StartupProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/mypath/actuator/health/liveness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 30, // Wait 5 minutes before declaring failure
			}))
		})


		It("should use default path when no context path is set", func()  {
			app.Spec.ContextPath = ""
			app.Spec.Port = 8000

			Expect(k8sClient.Update(ctx, app)).To(Succeed())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).NotTo(HaveOccurred())

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, deploy)).To(Succeed())

			container := deploy.Spec.Template.Spec.Containers[0]

			Expect(*container.ReadinessProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/actuator/health/readiness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}))

			Expect(*container.LivenessProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/actuator/health/liveness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}))

			Expect(*container.StartupProbe).To(Equal(corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(8000),
						Path: "/actuator/health/liveness",
						Scheme: "HTTP",
					},
				},
				InitialDelaySeconds: 0,
				TimeoutSeconds: 1,
				PeriodSeconds: 10,
				SuccessThreshold: 1,
				FailureThreshold: 30, // Wait 5 minutes before declaring failure
			}))
		})
	})

})
