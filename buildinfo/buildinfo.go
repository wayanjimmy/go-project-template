package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Service = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Service   string `json:"service"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func Get() Info {
	return Info{
		Version:   Version,
		Service:   Service,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func String() string {
	return fmt.Sprintf("service=%s version=%s go=%s",
		Service, Version, runtime.Version())
}
