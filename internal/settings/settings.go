package settings

import (
	"context"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	configMapName = "soyplane-cm"
	configKey     = "config.yaml"
)

var (
	cfg     *SoyplaneSettings
	cfgLock sync.RWMutex
)

type SoyplaneSettings struct {
	Execution struct {
		DefaultEngine struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		} `yaml:"defaultEngine"`
		DefaultImage string `yaml:"defaultImage"`
	} `yaml:"execution"`
}

// Get returns the current in-memory config.
func Get() *SoyplaneSettings {
	cfgLock.RLock()
	defer cfgLock.RUnlock()
	return cfg
}

// SettingsWatcher watches the soyplane-cm ConfigMap and triggers config reload on change.
type SettingsWatcher struct {
	client.Client
	Namespace string
}

func (cw *SettingsWatcher) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := ctrl.Log
	if req.Name != configMapName || req.Namespace != cw.Namespace {
		return reconcile.Result{}, nil
	}

	if err := cw.Reload(ctx); err != nil {
		log.Error(err, "Failed to reload settings from ConfigMap", "ConfigMap", req.Name)
		return reconcile.Result{}, err
	}

	log.Info("Settings reloaded from ConfigMap", "ConfigMap", req.Name)
	return reconcile.Result{}, nil
}

func (cw *SettingsWatcher) Reload(ctx context.Context) error {
	return cw.Load(ctx, nil)
}

// Reload fetches the latest ConfigMap and updates the in-memory config.
func (cw *SettingsWatcher) Load(ctx context.Context, reader *client.Reader) error {
	var cm corev1.ConfigMap
	var err error
	if reader != nil {
		err = (*reader).Get(ctx, types.NamespacedName{Name: configMapName, Namespace: cw.Namespace}, &cm)
	} else {
		err = cw.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: cw.Namespace}, &cm)
	}

	if err != nil {
		return fmt.Errorf("failed to get configmap: %w", err)
	}

	raw, ok := cm.Data[configKey]
	if !ok {
		return fmt.Errorf("configmap missing key %q", configKey)
	}

	var newCfg SoyplaneSettings
	if err := yaml.Unmarshal([]byte(raw), &newCfg); err != nil {
		return fmt.Errorf("failed to unmarshal soyplane config: %w", err)
	}

	cfgLock.Lock()
	cfg = &newCfg
	cfgLock.Unlock()
	return nil
}

func (cw *SettingsWatcher) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return e.Object.GetName() == configMapName && e.Object.GetNamespace() == cw.Namespace
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectNew.GetName() == configMapName && e.ObjectNew.GetNamespace() == cw.Namespace
			},
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		}).
		Named("soyplane_settings").
		Complete(cw)
}
