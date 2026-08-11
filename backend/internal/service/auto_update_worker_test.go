package service

import (
	"testing"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestAutoUpdateWorkerStartGating(t *testing.T) {
	t.Run("source build disables worker", func(t *testing.T) {
		updateSvc := &UpdateService{buildType: "source"}
		worker := NewAutoUpdateWorker(updateSvc, nil, &config.Config{})
		worker.Start()
		assert.False(t, worker.started, "source build must not start the worker")
		assert.True(t, worker.stopped)
		worker.Stop()
	})

	t.Run("auto_enabled=false disables worker", func(t *testing.T) {
		updateSvc := &UpdateService{buildType: "release"}
		cfg := &config.Config{}
		cfg.Update.AutoEnabled = false
		worker := NewAutoUpdateWorker(updateSvc, nil, cfg)
		worker.Start()
		assert.False(t, worker.started)
		assert.True(t, worker.stopped)
		worker.Stop()
	})

	t.Run("release build with auto enabled starts worker", func(t *testing.T) {
		updateSvc := &UpdateService{buildType: "release"}
		cfg := &config.Config{}
		cfg.Update.AutoEnabled = true
		cfg.Update.AutoIntervalHours = 1
		worker := NewAutoUpdateWorker(updateSvc, nil, cfg)
		worker.Start()
		assert.True(t, worker.started)
		assert.False(t, worker.stopped)
		worker.Stop()
	})
}
