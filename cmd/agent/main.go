package main

import (
	"flag"
	"log"
	"os"

	"github.com/soyplane-io/soyplane/internal/agent"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	name := os.Getenv("TOFU_EXECUTION_NAME")
	namespace := os.Getenv("TOFU_EXECUTION_NAMESPACE")
	if name == "" || namespace == "" {
		log.Fatal("Missing TOFU_EXECUTION_NAME or TOFU_EXECUTION_NAMESPACE")
	}

	if err := agent.Run(name, namespace); err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}
}
