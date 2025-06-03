/*
Copyright © 2025 acejilam acejilam@gmail.com
*/
package cmd

import (
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	extclient "github.com/ls-2018/k8s-watch/pkg/client"
	"github.com/ls-2018/k8s-watch/pkg/controller"
	"github.com/ls-2018/k8s-watch/pkg/hook_webhook"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/spf13/cobra"
)

// serverCmd represents the run command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Long-term service, monitoring changes",
	Long:  `Long-term service, monitoring changes`,
	Run: func(cmd *cobra.Command, args []string) {

		ctx := ctrl.SetupSignalHandler()

		cfg, err := rest.InClusterConfig()
		if err != nil {
			klog.Error(err, "unable to create in-cluster config")
			os.Exit(1)
		}
		cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return &LoggingTransport{rt: rt}
		}
		err = extclient.NewRegistry(cfg)
		if err != nil {
			klog.Error(err, "unable to init kruise clientset and informer")
			os.Exit(1)
		}

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: clientgoscheme.Scheme,
			WebhookServer: webhook.NewServer(webhook.Options{
				Host:     "0.0.0.0",
				Port:     8443,
				CertDir:  filepath.Dir(CertFile),
				CertName: filepath.Base(CertFile),
				KeyName:  filepath.Base(KeyFile),
			}),
		})
		if err != nil {
			klog.Error(err)
			os.Exit(1)
		}
		rand.Seed(time.Now().UnixNano())
		ctrl.SetLogger(klogr.New())
		go func() {
			if err = controller.SetupWithManager(mgr); err != nil {
				klog.Error(err, "unable to setup controllers")
				os.Exit(1)
			}
		}()
		hook_webhook.SetupWithManager(mgr)
		klog.Info("starting manager")
		if err := mgr.Start(ctx); err != nil {
			klog.Error(err, "problem running manager")
			os.Exit(1)
		}
	},
}

func init() {

	serverCmd.Flags().StringVar(&CertFile, "tls-cert-file", "/certs/server.pem", "File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert).")
	serverCmd.Flags().StringVar(&KeyFile, "tls-key-file", "/certs/key.pem", "File containing the default x509 private key matching --tls-cert-file.")

	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var CertFile string
var KeyFile string

type LoggingTransport struct {
	rt http.RoundTripper
}

func (l *LoggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	klog.Infoln(request.URL.String(), request.Method)
	return l.rt.RoundTrip(request)
}
