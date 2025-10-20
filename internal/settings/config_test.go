package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
)

// reset clears package-level globals so tests can start from a clean state.
func reset() {
	mu.Lock()
	defer mu.Unlock()

	K = koanf.New(".")
	providers = nil
	cached = SoyplaneSettings{}
	settingsReady = false
}

func TestExecutionBeforeInit(t *testing.T) {
	reset()

	if _, err := Execution(); !errors.Is(err, ErrSettingsNotLoaded) {
		t.Fatalf("expected ErrSettingsNotLoaded, got %v", err)
	}
}

func TestInitLoadsConfiguration(t *testing.T) {
	t.Cleanup(reset)
	reset()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "test: true\nexecution:\n  defaultImage: example/image:latest\n")

	if err := Init([]string{cfgPath}, false); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	value, err := Execution()
	if err != nil {
		t.Fatalf("Execution returned error: %v", err)
	}

	if value.DefaultImage != "example/image:latest" {
		t.Fatalf("expected execution.defaultImage to be %q, got %q", "example/image:latest", value.DefaultImage)
	}
}

func TestLoadPreservesPreviousSnapshotOnFailure(t *testing.T) {
	t.Cleanup(reset)
	reset()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "test: true\nexecution:\n  defaultImage: stable/image:v1\n")

	if err := Init([]string{cfgPath}, false); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	got, err := Execution()
	if err != nil {
		t.Fatalf("Execution returned error: %v", err)
	}

	if got.DefaultImage != "stable/image:v1" {
		t.Fatalf("expected initial defaultImage to be %q, got %q", "stable/image:v1", got.DefaultImage)
	}

	// Corrupt the file to force Load to fail.
	writeFile(t, cfgPath, "test: true\nexecution: [not: valid yaml")

	err = Load()
	if err == nil {
		t.Fatalf("expected Load to fail with invalid yaml")
	}

	// The in-memory snapshot should still have the original value.
	got, err = Execution()
	if err != nil {
		t.Fatalf("Execution returned error: %v", err)
	}

	if got.DefaultImage != "stable/image:v1" {
		t.Fatalf("expected defaultImage to remain %q after failed reload, got %q", "stable/image:v1", got.DefaultImage)
	}
}

func TestInitWatchFailureAllowsRetry(t *testing.T) {
	t.Cleanup(reset)
	reset()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "test: true\nexecution:\n  defaultImage: retry/image:v1\n")

	origFactory := providerFactory
	t.Cleanup(func() {
		providerFactory = origFactory
	})

	var calls int
	providerFactory = func(path string) koanf.Provider {
		calls++
		base := origFactory(path)
		if calls == 1 {
			return watchErrorProvider{base: base}
		}
		return base
	}

	if err := Init([]string{cfgPath}, true); err == nil {
		t.Fatalf("expected Init to fail when watch setup fails")
	}

	if _, err := Execution(); !errors.Is(err, ErrSettingsNotLoaded) {
		t.Fatalf("expected ErrSettingsNotLoaded after failed watch setup, got %v", err)
	}

	if err := Init([]string{cfgPath}, true); err != nil {
		t.Fatalf("Init retry returned error: %v", err)
	}

	cfg, err := Execution()
	if err != nil {
		t.Fatalf("Execution returned error: %v", err)
	}

	if cfg.DefaultImage != "retry/image:v1" {
		t.Fatalf("expected defaultImage to be %q, got %q", "retry/image:v1", cfg.DefaultImage)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

type watchErrorProvider struct {
	base koanf.Provider
}

func (p watchErrorProvider) ReadBytes() ([]byte, error) {
	return p.base.ReadBytes()
}

func (p watchErrorProvider) Read() (map[string]interface{}, error) {
	return p.base.Read()
}

func (p watchErrorProvider) Watch(func(any, error)) error {
	return errors.New("watch error")
}
