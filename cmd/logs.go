/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var template string
var group string
var version string
var resources string

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Display the observed data",
	Long:  `Display the observed data`,
	Run: func(cmd *cobra.Command, args []string) {
		h := NewHook()
		mu, err := h.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), ns, metav1.GetOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		scope := admissionregistrationv1.NamespacedScope
		mu.Webhooks[0].Rules = []admissionregistrationv1.RuleWithOperations{
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.OperationAll,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{group},
					APIVersions: []string{"v1", "v1beta1"},
					Resources: []string{
						resources,
						resources + "/*",
					},
					Scope: &scope,
				},
			},
		}
		_, err = h.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.Background(), mu, metav1.UpdateOptions{})
		if err != nil {
			klog.Fatal(err)
		}

		list, err := h.client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			klog.Fatal(err)
		}
		if len(list.Items) < 1 {
			klog.Fatal("no pods found")
		}
		pod := list.Items[0]
		pod.Annotations["template"] = template
		_, err = h.client.CoreV1().Pods(ns).Update(context.Background(), &pod, metav1.UpdateOptions{})
		if err != nil {
			klog.Fatal(err)
		}

		now := metav1.NewTime(time.Now())
		logs := h.client.CoreV1().Pods(ns).GetLogs(pod.Name, &corev1.PodLogOptions{
			Follow:    true,
			SinceTime: &now,
		})

		stream, err := logs.Stream(cmd.Context())
		if err != nil {
			klog.Fatal(err)
			return
		}
		defer stream.Close()

		buf := make([]byte, 1024*10*10)
		t := fmt.Sprintf("template:%s", base64.StdEncoding.EncodeToString([]byte(template)))
		for {
			n, err := stream.Read(buf)
			if err != nil {
				klog.Fatal(err)
				return
			}
			if n == 0 {
				break
			}

			if strings.Contains(string(buf[0:n]), t) {
				klog.Info(strings.SplitN(string(buf[0:n]), t, 2)[1])
			}
		}
	},
}

func init() {
	logsCmd.Flags().StringVar(&template, "template", "spec.replicas", "the corresponding resource attributes of the monitoring")
	logsCmd.Flags().StringVar(&group, "group", "apps", "group")
	logsCmd.Flags().StringVar(&version, "version", "v1", "group version ")
	logsCmd.Flags().StringVar(&resources, "resources", "deployments", "resource name")

	rootCmd.AddCommand(logsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
