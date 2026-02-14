package version

import (
	"fmt"
	"math/rand"
	"time"
)

// Build-time variables, injected via ldflags.
var (
	Version   = ""
	GitCommit = ""
	BuildDate = ""
)

// GetVersion returns the full version string.
// If built via `make build`, returns e.g. "v1.0.0-20260214-a3f1".
// Otherwise returns a dev version like "v1.0.0-20260214-dev".
func GetVersion() string {
	if Version != "" {
		return Version
	}
	return fmt.Sprintf("v1.0.0-%s-dev", time.Now().Format("20060102"))
}

// GenerateBuildVersion produces a version string for use in Makefile ldflags.
// Format: v1.0.0-YYYYMMDD-XXXX (4-char random hex suffix).
func GenerateBuildVersion() string {
	suffix := fmt.Sprintf("%04x", rand.Intn(0xFFFF))
	return fmt.Sprintf("v1.0.0-%s-%s", time.Now().Format("20060102"), suffix)
}
