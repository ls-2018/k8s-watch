/*
Copyright © 2025 acejilam acejilam@gmail.com
*/
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Clean up resources",
	Long:  `Clean up resources`,
	Run: func(cmd *cobra.Command, args []string) {
		h := NewHook()
		err := h.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), ns, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			klog.Error(err)
		}

		h.status.Start("Cleaning up resources")
		defer func() { h.status.End(h.err == nil) }()

		err = h.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), ns, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			h.err = err
			return
		}
		// speed up ✈️
		h.client.AppsV1().Deployments(ns).DeleteCollection(context.Background(), metav1.DeleteOptions{
			GracePeriodSeconds: func(i int64) *int64 { return &i }(0),
		}, metav1.ListOptions{
			LabelSelector: "app=k8s-resources-watch",
		})
		h.client.AppsV1().StatefulSets(ns).DeleteCollection(context.Background(), metav1.DeleteOptions{
			GracePeriodSeconds: func(i int64) *int64 { return &i }(0),
		}, metav1.ListOptions{
			LabelSelector: "app=k8s-resources-watch",
		})
		h.client.CoreV1().Pods(ns).DeleteCollection(context.Background(), metav1.DeleteOptions{
			GracePeriodSeconds: func(i int64) *int64 { return &i }(0),
		}, metav1.ListOptions{
			LabelSelector: "app=k8s-resources-watch",
		})

		for {
			time.Sleep(1 * time.Second)
			list, err := h.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("kubernetes.io/metadata.name=%s", ns),
			})
			if err != nil {
				h.err = fmt.Errorf("failed to watch ns: %w", err)
				return
			}

			if len(list.Items) == 0 {
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// destroyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// destroyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
