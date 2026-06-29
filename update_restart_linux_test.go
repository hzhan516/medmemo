//go:build linux

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestartAfterUpdate_StartsNewAppImage(t *testing.T) {
	tmpDir := t.TempDir()
	marker := filepath.Join(tmpDir, "restarted")
	script := filepath.Join(tmpDir, "MedMemo.AppImage")

	content := "#!/bin/sh\ntouch " + marker + "\n"
	require.NoError(t, os.WriteFile(script, []byte(content), 0755))

	err := restartAfterUpdate(context.Background(), script)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err := os.Stat(marker)
		return err == nil
	}, 2*time.Second, 100*time.Millisecond)
}

func TestRestartAfterUpdate_InvalidPathReturnsError(t *testing.T) {
	err := restartAfterUpdate(context.Background(), "/nonexistent/AppImage")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start new AppImage")
}
