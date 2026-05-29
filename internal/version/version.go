package version

// Version is replaced by release builds. Local development uses "dev".
var Version = "dev"

func String() string {
	return Version
}
