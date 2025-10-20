package settings

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	// K holds the active configuration tree.
	K *koanf.Koanf = koanf.New(".")
	// providers is the list of configured providers to load from.
	providers []koanf.Provider
	// cached holds the typed settings derived from K.
	cached        SoyplaneSettings
	settingsReady bool
	mu            sync.RWMutex
	logger        = ctrl.Log.WithName("settings")
)

var (
	eventMu       sync.RWMutex
	eventRecorder record.EventRecorder
	eventTarget   *corev1.ObjectReference
)

// DefaultConfigPaths is used when Init is passed an empty list.
var DefaultConfigPaths = []string{"config.yaml"}

var providerFactory = func(path string) koanf.Provider {
	return file.Provider(path)
}

// Init sets up providers from the given paths and loads the config once.
// If watch is true, file providers will be watched for changes and auto-reloaded.
func Init(paths []string, watch bool) error {
	mu.Lock()
	defer mu.Unlock()

	if settingsReady {
		logger.Info("config already initialized")
		return nil
	}

	if len(paths) == 0 {
		paths = DefaultConfigPaths
	}

	tmpProviders := make([]koanf.Provider, 0, len(paths))
	for _, p := range paths {
		tmpProviders = append(tmpProviders, providerFactory(p))
	}

	tree, cfg, err := loadProviders(tmpProviders)
	if err != nil {
		recordReloadFailure("init", err)
		return err
	}

	if watch {
		if err := attachWatchers(tmpProviders, paths); err != nil {
			recordReloadFailure("watch", err)
			return err
		}
	}

	providers = tmpProviders
	K = tree
	cached = cfg
	settingsReady = true

	if watch {
		logger.Info("config watch enabled", "paths", paths)
	}

	return nil
}

// Load forces a reload from all configured providers.
func Load() error {
	mu.Lock()
	defer mu.Unlock()
	return reloadLocked()
}

// reloadLocked reloads K from providers. Caller must hold mu.Lock().
func reloadLocked() error {
	tree, cfg, err := loadProviders(providers)
	if err != nil {
		recordReloadFailure("reload", err)
		return err
	}

	K = tree
	cached = cfg
	settingsReady = true
	return nil
}

func loadProviders(provs []koanf.Provider) (*koanf.Koanf, SoyplaneSettings, error) {
	tree := koanf.New(".")
	for _, prov := range provs {
		if err := tree.Load(prov, yaml.Parser()); err != nil {
			return nil, SoyplaneSettings{}, err
		}
	}
	cfg, err := hydrateSettings(tree)
	if err != nil {
		return nil, SoyplaneSettings{}, err
	}
	return tree, cfg, nil
}

// ConfigureEvents wires an event recorder and optional target for reporting setting failures.
func ConfigureEvents(rec record.EventRecorder, ref *corev1.ObjectReference) {
	eventMu.Lock()
	defer eventMu.Unlock()
	eventRecorder = rec
	eventTarget = ref
}

func attachWatchers(provs []koanf.Provider, paths []string) error {
	type watchable interface {
		Watch(func(event any, err error)) error
	}

	for i, prov := range provs {
		path := ""
		if i < len(paths) {
			path = paths[i]
		}
		wp, ok := prov.(watchable)
		if !ok {
			logger.Info("provider does not support watch", "path", path)
			continue
		}
		pathCopy := path
		if err := wp.Watch(func(event any, err error) {
			if err != nil {
				logger.Error(err, "config watch error", "path", pathCopy)
				return
			}
			logger.Info("config changed; reloading", "path", pathCopy)
			if err := Load(); err != nil {
				logger.Error(err, "config reload failed", "path", pathCopy)
			}
		}); err != nil {
			logger.Error(err, "watch setup failed", "path", path)
			return err
		}
	}
	return nil
}
