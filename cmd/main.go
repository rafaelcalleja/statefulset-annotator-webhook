package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
)

func annotateStatefulSetMutator(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
	statefulSet, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		// If not a statefulSet just continue the mutation chain(if there is one) and don't do nothing.
		return &kwhmutating.MutatorResult{}, nil
	}

	// Mutate our object with the required annotations.
	if statefulSet.Annotations == nil {
		statefulSet.Annotations = make(map[string]string)
	}

	// make sure source object allows replication
	_, ok = statefulSet.Spec.Template.ObjectMeta.Annotations["statefulset-annotate-webhook.dokify.dev"]
	if !ok {
		return &kwhmutating.MutatorResult{}, nil
	}

	prefix := "replicate-"
	for key, val := range statefulSet.Spec.Template.Annotations {
		indexOf := strings.Index(key, prefix)
		if indexOf >= 0 {
			statefulSet.Annotations[key[(indexOf+len(prefix)):]] = val
		}
	}

	return &kwhmutating.MutatorResult{
		MutatedObject: statefulSet,
	}, nil
}

type config struct {
	certFile string
	keyFile  string
}

func initFlags() *config {
	cfg := &config{}

	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fl.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	fl.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")

	_ = fl.Parse(os.Args[1:])
	return cfg
}

func main() {
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)

	cfg := initFlags()

	// Create our mutator
	mt := kwhmutating.MutatorFunc(annotateStatefulSetMutator)

	mcfg := kwhmutating.WebhookConfig{
		ID:      "statefulSetAnnotate",
		Obj:     &appsv1.StatefulSet{},
		Mutator: mt,
		Logger:  logger,
	}
	wh, err := kwhmutating.NewWebhook(mcfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		os.Exit(1)
	}

	// Get the handler for our webhook.
	whHandler, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: wh, Logger: logger})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook handler: %s", err)
		os.Exit(1)
	}
	logger.Infof("Listening on :8080")
	err = http.ListenAndServeTLS(":8080", cfg.certFile, cfg.keyFile, whHandler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serving webhook: %s", err)
		os.Exit(1)
	}
}
