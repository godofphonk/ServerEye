package version

// Version information
const (
	// Version is the current version of ServerEye
	Version = "1.0.4"

	// BuildDate is set during build time
	BuildDate = "dev"

	// GitCommit is set during build time
	GitCommit = "dev"
)

// GetVersion returns the full version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns version with build info
func GetFullVersion() string {
	if BuildDate == "dev" {
		return Version + "-dev"
	}
	return Version
}
