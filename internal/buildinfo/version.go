package buildinfo

// Version is the single source of truth for tool version metadata.
// Release builds override via -ldflags "-X github.com/afelin/cyberready/internal/buildinfo.Version=...".
var Version = "0.4.1"
