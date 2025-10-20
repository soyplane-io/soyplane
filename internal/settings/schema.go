package settings

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/v2"
	defaults "github.com/mcuadros/go-defaults"
)

// SoyplaneSettings captures all configuration knobs surfaced to runtime components.
type SoyplaneSettings struct {
	Execution ExecutionSettings `koanf:"execution" validate:"required"`
	Test      bool              `koanf:"test" validate:"required"`
}

// ExecutionSettings groups execution-specific knobs.
type ExecutionSettings struct {
	DefaultImage string `koanf:"defaultImage" default:"tofuutils/tenv:latest" validate:"required"`
}

// ErrSettingsNotLoaded indicates that settings have not been loaded yet.
var ErrSettingsNotLoaded = errors.New("settings not loaded")

var validate = validator.New(validator.WithRequiredStructEnabled())

// ConfigError surfaces validation issues for configuration fields.
type ConfigError struct {
	Field string
	Tag   string
	Value interface{}
}

func (e ConfigError) Error() string {
	return fmt.Sprintf("settings validation failed for %s (tag: %s)", e.Field, e.Tag)
}

func hydrateSettings(tree *koanf.Koanf) (SoyplaneSettings, error) {
	cfg := SoyplaneSettings{}

	if tree != nil {
		if err := tree.Unmarshal("", &cfg); err != nil {
			return SoyplaneSettings{}, err
		}
	}

	defaults.SetDefaults(&cfg)

	if err := validate.Struct(cfg); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) && len(ve) > 0 {
			first := ve[0]
			return SoyplaneSettings{}, ConfigError{
				Field: first.StructNamespace(),
				Tag:   first.Tag(),
				Value: first.Value(),
			}
		}
		return SoyplaneSettings{}, err
	}

	return cfg, nil
}

// Snapshot returns the cached settings snapshot.
func Snapshot() (SoyplaneSettings, error) {
	mu.RLock()
	defer mu.RUnlock()

	if !settingsReady {
		return SoyplaneSettings{}, ErrSettingsNotLoaded
	}

	return cached, nil
}

// Execution returns execution settings with defaults applied.
func Execution() (ExecutionSettings, error) {
	snap, err := Snapshot()
	if err != nil {
		return ExecutionSettings{}, err
	}

	return snap.Execution, nil
}
