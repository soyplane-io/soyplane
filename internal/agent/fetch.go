package agent

import (
	"context"

	opentofuv1alpha1 "github.com/soyplane-io/soyplane/api/opentofu/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func FetchTofuExecution(ctx context.Context, cl ctrlclient.Client, name, namespace string) (*opentofuv1alpha1.TofuExecution, error) {
	var exec opentofuv1alpha1.TofuExecution
	err := cl.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &exec)
	return &exec, err
}

func FetchTofuModule(ctx context.Context, cl ctrlclient.Client, name, namespace string) (*opentofuv1alpha1.TofuModule, error) {
	var module opentofuv1alpha1.TofuModule
	err := cl.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &module)
	return &module, err
}
