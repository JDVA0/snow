package blizzard

import (
	"runtime"
	"testing"
)

func TestSupportedPlatform(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		// Supported desktop platforms.
	default:
		t.Skipf("platform %s is outside the compatibility matrix", runtime.GOOS)
	}
}
