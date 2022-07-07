package manifests

import (
	"fmt"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"

	configv1 "github.com/grafana/loki/operator/apis/config/v1"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/grafana/loki/operator/internal/manifests/openshift"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestApplyGatewayDefaultsOptions(t *testing.T) {
	type tt struct {
		desc string
		opts *Options
		want *Options
	}

	tc := []tt{
		{
			desc: "static mode",
			opts: &Options{
				Stack: lokiv1.LokiStackSpec{
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.Static,
					},
				},
			},
			want: &Options{
				Stack: lokiv1.LokiStackSpec{
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.Static,
					},
				},
			},
		},
		{
			desc: "dynamic mode",
			opts: &Options{
				Stack: lokiv1.LokiStackSpec{
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.Dynamic,
					},
				},
			},
			want: &Options{
				Stack: lokiv1.LokiStackSpec{
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.Dynamic,
					},
				},
			},
		},
		{
			desc: "openshift-logging mode",
			opts: &Options{
				Name:              "lokistack-ocp",
				Namespace:         "stack-ns",
				GatewayBaseDomain: "example.com",
				Stack: lokiv1.LokiStackSpec{
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.OpenshiftLogging,
					},
				},
				Tenants: Tenants{
					Configs: map[string]TenantConfig{
						"application": {
							OpenShift: &TenantOpenShiftSpec{
								CookieSecret: "D31SJpSmPe6aUDTtU2zqAoW1gqEKoH5T",
							},
						},
						"infrastructure": {
							OpenShift: &TenantOpenShiftSpec{
								CookieSecret: "i3N1paUy9JwNZIktni4kqXPuMvIHtHNe",
							},
						},
						"audit": {
							OpenShift: &TenantOpenShiftSpec{
								CookieSecret: "6UssDXle7OHElqSW4M0DNRZ6JbaTjDM3",
							},
						},
					},
				},
			},
			want: &Options{
				Name:              "lokistack-ocp",
				Namespace:         "stack-ns",
				GatewayBaseDomain: "example.com",
				Stack: lokiv1.LokiStackSpec{
					Tenants: &lokiv1.TenantsSpec{
						Mode: lokiv1.OpenshiftLogging,
					},
				},
				Tenants: Tenants{
					Configs: map[string]TenantConfig{
						"application": {
							OpenShift: &TenantOpenShiftSpec{
								CookieSecret: "D31SJpSmPe6aUDTtU2zqAoW1gqEKoH5T",
							},
						},
						"infrastructure": {
							OpenShift: &TenantOpenShiftSpec{
								CookieSecret: "i3N1paUy9JwNZIktni4kqXPuMvIHtHNe",
							},
						},
						"audit": {
							OpenShift: &TenantOpenShiftSpec{
								CookieSecret: "6UssDXle7OHElqSW4M0DNRZ6JbaTjDM3",
							},
						},
					},
				},
				OpenShiftOptions: openshift.Options{
					BuildOpts: openshift.BuildOptions{
						LokiStackName:        "lokistack-ocp",
						LokiStackNamespace:   "stack-ns",
						GatewayName:          "lokistack-ocp-gateway",
						GatewaySvcName:       "lokistack-ocp-gateway-http",
						GatewaySvcTargetPort: "public",
						Labels:               ComponentLabels(LabelGatewayComponent, "lokistack-ocp"),
					},
					Authentication: []openshift.AuthenticationSpec{
						{
							TenantName:     "application",
							TenantID:       "",
							ServiceAccount: "lokistack-ocp-gateway",
							RedirectURL:    "https://lokistack-ocp-stack-ns.apps.example.com/openshift/application/callback",
						},
						{
							TenantName:     "infrastructure",
							TenantID:       "",
							ServiceAccount: "lokistack-ocp-gateway",
							RedirectURL:    "https://lokistack-ocp-stack-ns.apps.example.com/openshift/infrastructure/callback",
						},
						{
							TenantName:     "audit",
							TenantID:       "",
							ServiceAccount: "lokistack-ocp-gateway",
							RedirectURL:    "https://lokistack-ocp-stack-ns.apps.example.com/openshift/audit/callback",
						},
					},
					Authorization: openshift.AuthorizationSpec{
						OPAUrl: "http://localhost:8082/v1/data/lokistack/allow",
					},
				},
			},
		},
	}
	for _, tc := range tc {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			err := ApplyGatewayDefaultOptions(tc.opts)
			require.NoError(t, err)

			for i, a := range tc.opts.OpenShiftOptions.Authentication {
				require.NotEmpty(t, a.TenantID)
				require.NotEmpty(t, a.CookieSecret)
				require.Len(t, a.CookieSecret, 32)

				a.TenantID = ""
				a.CookieSecret = ""
				tc.opts.OpenShiftOptions.Authentication[i] = a
				tc.opts.OpenShiftOptions.Authentication[i] = a
			}

			require.Equal(t, tc.want, tc.opts)
		})
	}
}

func TestConfigureDeploymentForMode(t *testing.T) {
	type tt struct {
		desc         string
		mode         lokiv1.ModeType
		stackName    string
		stackNs      string
		featureGates configv1.FeatureGates
		dpl          *appsv1.Deployment
		want         *appsv1.Deployment
	}

	tc := []tt{
		{
			desc: "static mode",
			mode: lokiv1.Static,
			dpl:  &appsv1.Deployment{},
			want: &appsv1.Deployment{},
		},
		{
			desc: "dynamic mode",
			mode: lokiv1.Dynamic,
			dpl:  &appsv1.Deployment{},
			want: &appsv1.Deployment{},
		},
		{

			desc:      "openshift-logging mode",
			mode:      lokiv1.OpenshiftLogging,
			stackName: "test",
			stackNs:   "test-ns",
			dpl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: gatewayContainerName,
									Args: []string{
										"--logs.read.endpoint=http://example.com",
										"--logs.tail.endpoint=http://example.com",
										"--logs.write.endpoint=http://example.com",
										fmt.Sprintf("--web.healthchecks.url=http://localhost:%d", gatewayHTTPPort),
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: gatewayContainerName,
									Args: []string{
										"--logs.read.endpoint=http://example.com",
										"--logs.tail.endpoint=http://example.com",
										"--logs.write.endpoint=http://example.com",
										fmt.Sprintf("--web.healthchecks.url=https://localhost:%d", gatewayHTTPPort),
										"--tls.server.cert-file=/var/run/tls/http/tls.crt",
										"--tls.server.key-file=/var/run/tls/http/tls.key",
										"--tls.healthchecks.server-ca-file=/var/run/ca/service-ca.crt",
										fmt.Sprintf("--tls.healthchecks.server-name=%s", "test-gateway-http.test-ns.svc.cluster.local"),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      tlsSecretVolume,
											ReadOnly:  true,
											MountPath: httpTLSDir,
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTPS,
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTPS,
											},
										},
									},
								},
								{
									Name:  "opa",
									Image: "quay.io/observatorium/opa-openshift:latest",
									Args: []string{
										"--log.level=warn",
										"--opa.package=lokistack",
										"--opa.matcher=kubernetes_namespace_name",
										"--web.listen=:8082",
										"--web.internal.listen=:8083",
										"--web.healthchecks.url=http://localhost:8082",
										`--openshift.mappings=application=loki.grafana.com`,
										`--openshift.mappings=infrastructure=loki.grafana.com`,
										`--openshift.mappings=audit=loki.grafana.com`,
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          openshift.GatewayOPAHTTPPortName,
											ContainerPort: openshift.GatewayOPAHTTPPort,
											Protocol:      corev1.ProtocolTCP,
										},
										{
											Name:          openshift.GatewayOPAInternalPortName,
											ContainerPort: openshift.GatewayOPAInternalPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/live",
												Port:   intstr.FromInt(int(openshift.GatewayOPAInternalPort)),
												Scheme: corev1.URISchemeHTTP,
											},
										},
										TimeoutSeconds:   2,
										PeriodSeconds:    30,
										FailureThreshold: 10,
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/ready",
												Port:   intstr.FromInt(int(openshift.GatewayOPAInternalPort)),
												Scheme: corev1.URISchemeHTTP,
											},
										},
										TimeoutSeconds:   1,
										PeriodSeconds:    5,
										FailureThreshold: 12,
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: tlsSecretVolume,
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "test-gateway-http-tls",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:      "openshift-logging mode with-tls-service-monitor-config",
			mode:      lokiv1.OpenshiftLogging,
			stackName: "test",
			stackNs:   "test-ns",
			featureGates: configv1.FeatureGates{
				HTTPEncryption:             true,
				ServiceMonitorTLSEndpoints: true,
			},
			dpl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: gatewayContainerName,
									Args: []string{
										"--logs.read.endpoint=http://example.com",
										"--logs.tail.endpoint=http://example.com",
										"--logs.write.endpoint=http://example.com",
										fmt.Sprintf("--web.healthchecks.url=http://localhost:%d", gatewayHTTPPort),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      tlsSecretVolume,
											ReadOnly:  true,
											MountPath: httpTLSDir,
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
											},
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: tlsSecretVolume,
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "test-gateway-http-tls",
										},
									},
								},
							},
						},
					},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: gatewayContainerName,
									Args: []string{
										"--logs.read.endpoint=http://example.com",
										"--logs.tail.endpoint=http://example.com",
										"--logs.write.endpoint=http://example.com",
										fmt.Sprintf("--web.healthchecks.url=https://localhost:%d", gatewayHTTPPort),
										"--tls.server.cert-file=/var/run/tls/http/tls.crt",
										"--tls.server.key-file=/var/run/tls/http/tls.key",
										"--tls.healthchecks.server-ca-file=/var/run/ca/service-ca.crt",
										fmt.Sprintf("--tls.healthchecks.server-name=%s", "test-gateway-http.test-ns.svc.cluster.local"),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      tlsSecretVolume,
											ReadOnly:  true,
											MountPath: httpTLSDir,
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTPS,
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTPS,
											},
										},
									},
								},
								{
									Name:  "opa",
									Image: "quay.io/observatorium/opa-openshift:latest",
									Args: []string{
										"--log.level=warn",
										"--opa.package=lokistack",
										"--opa.matcher=kubernetes_namespace_name",
										"--web.listen=:8082",
										"--web.internal.listen=:8083",
										"--web.healthchecks.url=http://localhost:8082",
										"--tls.internal.server.cert-file=/var/run/tls/http/tls.crt",
										"--tls.internal.server.key-file=/var/run/tls/http/tls.key",
										`--openshift.mappings=application=loki.grafana.com`,
										`--openshift.mappings=infrastructure=loki.grafana.com`,
										`--openshift.mappings=audit=loki.grafana.com`,
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          openshift.GatewayOPAHTTPPortName,
											ContainerPort: openshift.GatewayOPAHTTPPort,
											Protocol:      corev1.ProtocolTCP,
										},
										{
											Name:          openshift.GatewayOPAInternalPortName,
											ContainerPort: openshift.GatewayOPAInternalPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/live",
												Port:   intstr.FromInt(int(openshift.GatewayOPAInternalPort)),
												Scheme: corev1.URISchemeHTTPS,
											},
										},
										TimeoutSeconds:   2,
										PeriodSeconds:    30,
										FailureThreshold: 10,
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/ready",
												Port:   intstr.FromInt(int(openshift.GatewayOPAInternalPort)),
												Scheme: corev1.URISchemeHTTPS,
											},
										},
										TimeoutSeconds:   1,
										PeriodSeconds:    5,
										FailureThreshold: 12,
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      tlsSecretVolume,
											ReadOnly:  true,
											MountPath: httpTLSDir,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: tlsSecretVolume,
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "test-gateway-http-tls",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:      "openshift-logging mode with-cert-signing-service",
			mode:      lokiv1.OpenshiftLogging,
			stackName: "test",
			stackNs:   "test-ns",
			featureGates: configv1.FeatureGates{
				HTTPEncryption:             true,
				ServiceMonitorTLSEndpoints: true,
				OpenShift: configv1.OpenShiftFeatureGates{
					ServingCertsService: true,
				},
			},
			dpl: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gateway",
					Namespace: "test-ns",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: gatewayContainerName,
									Args: []string{
										"--other.args=foo-bar",
										"--logs.read.endpoint=http://example.com",
										"--logs.tail.endpoint=http://example.com",
										"--logs.write.endpoint=http://example.com",
										fmt.Sprintf("--web.healthchecks.url=http://localhost:%d", gatewayHTTPPort),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "tls-secret",
											ReadOnly:  true,
											MountPath: "/var/run/tls/http",
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTP,
											},
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "tls-secret-volume",
								},
							},
						},
					},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gateway",
					Namespace: "test-ns",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: "gateway",
							Containers: []corev1.Container{
								{
									Name: gatewayContainerName,
									Args: []string{
										"--other.args=foo-bar",
										"--logs.read.endpoint=https://example.com",
										"--logs.tail.endpoint=https://example.com",
										"--logs.write.endpoint=https://example.com",
										fmt.Sprintf("--web.healthchecks.url=https://localhost:%d", gatewayHTTPPort),
										"--logs.tls.ca-file=/var/run/ca/service-ca.crt",
										"--tls.server.cert-file=/var/run/tls/http/tls.crt",
										"--tls.server.key-file=/var/run/tls/http/tls.key",
										"--tls.healthchecks.server-ca-file=/var/run/ca/service-ca.crt",
										fmt.Sprintf("--tls.healthchecks.server-name=%s", "test-gateway-http.test-ns.svc.cluster.local"),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "tls-secret",
											ReadOnly:  true,
											MountPath: "/var/run/tls/http",
										},
										{
											Name:      "test-ca-bundle",
											ReadOnly:  true,
											MountPath: "/var/run/ca",
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTPS,
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Scheme: corev1.URISchemeHTTPS,
											},
										},
									},
								},
								{
									Name:  "opa",
									Image: "quay.io/observatorium/opa-openshift:latest",
									Args: []string{
										"--log.level=warn",
										"--opa.package=lokistack",
										"--opa.matcher=kubernetes_namespace_name",
										"--web.listen=:8082",
										"--web.internal.listen=:8083",
										"--web.healthchecks.url=http://localhost:8082",
										"--tls.internal.server.cert-file=/var/run/tls/http/tls.crt",
										"--tls.internal.server.key-file=/var/run/tls/http/tls.key",
										`--openshift.mappings=application=loki.grafana.com`,
										`--openshift.mappings=infrastructure=loki.grafana.com`,
										`--openshift.mappings=audit=loki.grafana.com`,
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          openshift.GatewayOPAHTTPPortName,
											ContainerPort: openshift.GatewayOPAHTTPPort,
											Protocol:      corev1.ProtocolTCP,
										},
										{
											Name:          openshift.GatewayOPAInternalPortName,
											ContainerPort: openshift.GatewayOPAInternalPort,
											Protocol:      corev1.ProtocolTCP,
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/live",
												Port:   intstr.FromInt(int(openshift.GatewayOPAInternalPort)),
												Scheme: corev1.URISchemeHTTPS,
											},
										},
										TimeoutSeconds:   2,
										PeriodSeconds:    30,
										FailureThreshold: 10,
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path:   "/ready",
												Port:   intstr.FromInt(int(openshift.GatewayOPAInternalPort)),
												Scheme: corev1.URISchemeHTTPS,
											},
										},
										TimeoutSeconds:   1,
										PeriodSeconds:    5,
										FailureThreshold: 12,
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      tlsSecretVolume,
											ReadOnly:  true,
											MountPath: httpTLSDir,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "tls-secret-volume",
								},
								{
									Name: "test-ca-bundle",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											DefaultMode: &defaultConfigMapMode,
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "test-ca-bundle",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tc {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			err := configureDeploymentForMode(tc.dpl, tc.mode, tc.featureGates, tc.stackName, tc.stackNs)
			require.NoError(t, err)
			require.Equal(t, tc.want, tc.dpl)
		})
	}
}

func TestConfigureServiceForMode(t *testing.T) {
	type tt struct {
		desc string
		mode lokiv1.ModeType
		svc  *corev1.ServiceSpec
		want *corev1.ServiceSpec
	}

	tc := []tt{
		{
			desc: "static mode",
			mode: lokiv1.Static,
			svc:  &corev1.ServiceSpec{},
			want: &corev1.ServiceSpec{},
		},
		{
			desc: "dynamic mode",
			mode: lokiv1.Dynamic,
			svc:  &corev1.ServiceSpec{},
			want: &corev1.ServiceSpec{},
		},
		{
			desc: "openshift-logging mode",
			mode: lokiv1.OpenshiftLogging,
			svc:  &corev1.ServiceSpec{},
			want: &corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: openshift.GatewayOPAInternalPortName,
						Port: openshift.GatewayOPAInternalPort,
					},
				},
			},
		},
	}
	for _, tc := range tc {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			err := configureServiceForMode(tc.svc, tc.mode)
			require.NoError(t, err)
			require.Equal(t, tc.want, tc.svc)
		})
	}
}

func TestConfigureServiceMonitorForMode(t *testing.T) {
	type tt struct {
		desc         string
		mode         lokiv1.ModeType
		featureGates configv1.FeatureGates
		sm           *monitoringv1.ServiceMonitor
		want         *monitoringv1.ServiceMonitor
	}

	tc := []tt{
		{
			desc: "static mode",
			mode: lokiv1.Static,
			sm:   &monitoringv1.ServiceMonitor{},
			want: &monitoringv1.ServiceMonitor{},
		},
		{
			desc: "dynamic mode",
			mode: lokiv1.Dynamic,
			sm:   &monitoringv1.ServiceMonitor{},
			want: &monitoringv1.ServiceMonitor{},
		},
		{
			desc: "openshift-logging mode",
			mode: lokiv1.OpenshiftLogging,
			sm:   &monitoringv1.ServiceMonitor{},
			want: &monitoringv1.ServiceMonitor{
				Spec: monitoringv1.ServiceMonitorSpec{
					Endpoints: []monitoringv1.Endpoint{
						{
							Port:   openshift.GatewayOPAInternalPortName,
							Path:   "/metrics",
							Scheme: "http",
						},
					},
				},
			},
		},
		{
			desc: "openshift-logging mode with-tls-service-monitor-config",
			mode: lokiv1.OpenshiftLogging,
			featureGates: configv1.FeatureGates{
				HTTPEncryption:             true,
				ServiceMonitorTLSEndpoints: true,
			},
			sm: &monitoringv1.ServiceMonitor{
				Spec: monitoringv1.ServiceMonitorSpec{
					Endpoints: []monitoringv1.Endpoint{
						{
							TLSConfig: &monitoringv1.TLSConfig{
								CAFile:   "/path/to/ca/file",
								CertFile: "/path/to/cert/file",
								KeyFile:  "/path/to/key/file",
							},
						},
					},
				},
			},
			want: &monitoringv1.ServiceMonitor{
				Spec: monitoringv1.ServiceMonitorSpec{
					Endpoints: []monitoringv1.Endpoint{
						{
							TLSConfig: &monitoringv1.TLSConfig{
								CAFile:   "/path/to/ca/file",
								CertFile: "/path/to/cert/file",
								KeyFile:  "/path/to/key/file",
							},
						},
						{
							Port:            openshift.GatewayOPAInternalPortName,
							Path:            "/metrics",
							Scheme:          "https",
							BearerTokenFile: BearerTokenFile,
							TLSConfig: &monitoringv1.TLSConfig{
								CAFile:   "/path/to/ca/file",
								CertFile: "/path/to/cert/file",
								KeyFile:  "/path/to/key/file",
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tc {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			err := configureServiceMonitorForMode(tc.sm, tc.mode, tc.featureGates)
			require.NoError(t, err)
			require.Equal(t, tc.want, tc.sm)
		})
	}
}
