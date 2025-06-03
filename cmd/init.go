/*
Copyright © 2025 acejilam acejilam@gmail.com
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/ls-2018/k8s-watch/pkg/logs"
	"github.com/ls-2018/k8s-watch/pkg/util/generator"
	"github.com/ls-2018/k8s-watch/pkg/util/writer"
	"github.com/spf13/cobra"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/kind/pkg/log"
)

var opt = Option{}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the resources related to the watch",
	Long:  `Initialize the resources related to the watch`,
	Run: func(cmd *cobra.Command, args []string) {

		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.Fatal(err)
		}

		client, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			klog.Fatal(err)
		}

		logger := logs.NewStdoutLogger(int32(opt.Verbosity))
		h := Hook{
			opt:    opt,
			client: client,
			logger: logger,
			status: logs.StatusForLogger(logger),
		}
		h.init()
		//h.destroy()
	},
}

func init() {
	initCmd.Flags().StringVar(&opt.ImageName, "image", "registry.cn-hangzhou.aliyuncs.com/ls-2018/k8s-watch-server:latest", "image name")
	initCmd.Flags().Int64Var(&opt.Verbosity, "verbosity", 1, "log verbosity level")
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type Option struct {
	ImageName string
	Verbosity int64
}

const ns = "ls-k8s-watch"

const serviceName = "ls-k8s-watch"

type K8sResources struct {
	Ns      *v1.Namespace
	SA      *v1.ServiceAccount
	Service *v1.Service
	//ClusterRole        *rbacv1.ClusterRole
	//ClusterRoleBinding *rbacv1.ClusterRoleBinding
	MutatingWebhook     *admissionregistrationv1.MutatingWebhookConfiguration
	Deployment          *appsv1.Deployment
	ClusterRole         *rbacv1.ClusterRole
	ClusterRoleBindings *rbacv1.ClusterRoleBinding
}

type Hook struct {
	client    *kubernetes.Clientset
	err       error
	status    *logs.Status
	logger    log.Logger
	resources K8sResources
	opt       Option
	certs     *generator.Artifacts
}

func NewHook() *Hook {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatal(err)
	}
	logger := logs.NewStdoutLogger(int32(opt.Verbosity))
	h := &Hook{
		opt:    opt,
		client: client,
		logger: logger,
		status: logs.StatusForLogger(logger),
	}
	return h
}

func (h *Hook) init() {
	h.logger.V(0).Info("Creating watch ...")
	h.prepareCert()
	h.initAdmissionWebhook()
	h.initNs()
	h.initSa()
	h.initClusterRole()
	h.initClusterRoleBinding()
	h.initService()
	h.initDeployment()
	h.watch()
	if h.err != nil {
		klog.Fatalf("failed to init watch: %v", h.err)
		return
	}
}

func (h *Hook) initNs() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing Namespace %s 📦", ns))
	defer func() { h.status.End(h.err == nil) }()
	obj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	controllerutil.SetOwnerReference(h.resources.MutatingWebhook, obj, scheme.Scheme)
	_, err := h.client.CoreV1().Namespaces().Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
		return
	}

	_ns, err := h.client.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.Ns = _ns
}

func (h *Hook) initSa() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing ServiceAccount %s 👷", ns))
	defer func() { h.status.End(h.err == nil) }()
	_, err := h.client.CoreV1().ServiceAccounts(ns).Create(context.TODO(), &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: h.resources.Ns.Name,
		},
	}, metav1.CreateOptions{})

	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
	}

	sa, err := h.client.CoreV1().ServiceAccounts(ns).Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.SA = sa
}

func (h *Hook) initClusterRole() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing ClusterRoles %s 🛡️", ns))
	defer func() { h.status.End(h.err == nil) }()
	obj := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	controllerutil.SetOwnerReference(h.resources.MutatingWebhook, obj, scheme.Scheme)

	_, err := h.client.RbacV1().ClusterRoles().Create(context.TODO(), obj, metav1.CreateOptions{})

	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
	}

	cr, err := h.client.RbacV1().ClusterRoles().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.ClusterRole = cr
}

func (h *Hook) initClusterRoleBinding() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing ClusterRoleBindings %s 🔀", ns))
	defer func() { h.status.End(h.err == nil) }()
	obj := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      ns,
				Namespace: h.resources.SA.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ns,
		},
	}
	controllerutil.SetOwnerReference(h.resources.MutatingWebhook, obj, scheme.Scheme)

	_, err := h.client.RbacV1().ClusterRoleBindings().Create(context.TODO(), obj, metav1.CreateOptions{})

	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
	}

	crb, err := h.client.RbacV1().ClusterRoleBindings().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.ClusterRoleBindings = crb
}

func (h *Hook) initService() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing Services %s 🎯", ns))
	defer func() { h.status.End(h.err == nil) }()
	_, err := h.client.CoreV1().Services(h.resources.Ns.Name).Create(context.TODO(), &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "k8s-resources-watch",
			},
			Ports: []v1.ServicePort{
				{
					Name:       "https",
					Protocol:   v1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt32(8443),
				},
			},
			Type: v1.ServiceTypeClusterIP,
		},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
		return
	}
	svc, err := h.client.CoreV1().Services(h.resources.Ns.Name).Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.Service = svc
}

func (h *Hook) initAdmissionWebhook() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing MutatingWebhookConfigurations %s 🛠️", ns))
	defer func() { h.status.End(h.err == nil) }()
	policy := admissionregistrationv1.Ignore
	scope := admissionregistrationv1.NamespacedScope
	se := admissionregistrationv1.SideEffectClassNone
	match := admissionregistrationv1.Equivalent
	obj := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "dy.watch.com",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      serviceName,
						Namespace: ns,
						Path:      pointer.String("/watch"),
						Port:      pointer.Int32(443),
					},
					CABundle: h.certs.CACert,
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					admissionregistrationv1.RuleWithOperations{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"apps"},
							APIVersions: []string{"v1"},
							Resources: []string{
								"deployments",
								"deployments/*",
							},
							Scope: &scope,
						},
					},
				},
				MatchPolicy:   &match,
				FailurePolicy: &policy,
				SideEffects:   &se,
				ObjectSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						metav1.LabelSelectorRequirement{
							Key:      "k8s-resources-watch",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
					},
				},
				AdmissionReviewVersions: []string{
					"v1",
				},
			},
		},
	}

	_, err := h.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
		return
	}

	mu, err := h.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), obj.Name, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.MutatingWebhook = mu
}

func (h *Hook) initDeployment() {
	if h.err != nil {
		return
	}
	h.status.Start(fmt.Sprintf("Preparing Deployments %s 🔁", ns))
	defer func() { h.status.End(h.err == nil) }()
	obj := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: map[string]string{
				"app":                 "k8s-resources-watch",
				"k8s-resources-watch": "k8s-resources-watch",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "k8s-resources-watch",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "k8s-resources-watch",
					},
					Annotations: map[string]string{
						"ca.crt":     string(h.certs.CACert),
						"server.pem": string(h.certs.Cert),
						"key.pem":    string(h.certs.Key),
						"template":   "spec.replicas",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "k8s-hook",
							Image: h.opt.ImageName,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "certs",
									MountPath: "/certs",
								},
							},
							//Command: []string{"/bin/bash", "-c", "sleep 1d"},
							Args: []string{
								"server",
							},
							ImagePullPolicy: v1.PullAlways,
						},
					},
					Volumes: []v1.Volume{
						v1.Volume{
							Name: "certs",
							VolumeSource: v1.VolumeSource{
								DownwardAPI: &v1.DownwardAPIVolumeSource{
									Items: []v1.DownwardAPIVolumeFile{
										v1.DownwardAPIVolumeFile{
											Path: "ca.crt",
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.annotations['ca.crt']",
											},
											Mode: pointer.Int32(0666),
										},
										v1.DownwardAPIVolumeFile{
											Path: "key.pem",
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.annotations['key.pem']",
											},
											Mode: pointer.Int32(0666),
										},
										v1.DownwardAPIVolumeFile{
											Path: "server.pem",
											FieldRef: &v1.ObjectFieldSelector{
												FieldPath: "metadata.annotations['server.pem']",
											},
											Mode: pointer.Int32(0666),
										},
									},
								},
							},
						},
					},
					ServiceAccountName: ns,
				},
			},
		},
	}

	_, err := h.client.AppsV1().Deployments(h.resources.Ns.Name).Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		h.err = err
		return
	}

	dp, err := h.client.AppsV1().Deployments(h.resources.Ns.Name).Get(context.TODO(), obj.Name, metav1.GetOptions{})
	if err != nil {
		h.err = err
	}
	h.resources.Deployment = dp
}

func (h *Hook) prepareCert() {
	certWriter, err := writer.NewFSCertWriter(writer.FSCertWriterOptions{
		Path: os.TempDir(),
	})
	if err != nil {
		h.err = fmt.Errorf("failed to create cert writer: %w", err)
		return
	}

	certs, _, err := certWriter.EnsureCert(fmt.Sprintf("%s.%s.svc", serviceName, ns))

	if err != nil {
		h.err = fmt.Errorf("failed to ensure certs: %w", err)
		return
	}
	h.certs = certs
}

func (h *Hook) watch() {
	if h.err != nil {
		return
	}
	h.status.Start("Watching Pods 🟣")
	defer func() { h.status.End(h.err == nil) }()
	list, err := h.client.CoreV1().Pods(h.resources.Ns.Name).Watch(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=k8s-resources-watch",
	})
	if err != nil {
		h.err = fmt.Errorf("failed to watch pods: %w", err)
		return
	}
	for event := range list.ResultChan() {
		pod, ok := event.Object.(*v1.Pod)
		if !ok {
			h.err = fmt.Errorf("unexpected type %T", event.Object)
			return
		}
		switch event.Type {
		case "MODIFIED":
			if pod.Status.Phase == v1.PodRunning {
				return
			}
		default:
		}
	}

}
