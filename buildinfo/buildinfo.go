package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
	Service   = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	Service   string `json:"service"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		Service:   Service,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func String() string {
	return fmt.Sprintf("service=%s version=%s commit=%s built=%s go=%s",
		Service, Version, Commit, BuildDate, runtime.Version())
}
