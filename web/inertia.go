package web

import (
	"encoding/json"
	"io/fs"

	inertia "github.com/petaki/inertia-go"
	"go-project-template/vite"
)

type InertiaConfig struct {
	Dev             bool
	RootTemplate    string
	RootTemplateDev string
	URL             string
	Version         string
	FS              fs.FS
	ManifestPath    string
	ManifestFS      fs.FS
}

func NewInertia(cfg InertiaConfig) *inertia.Inertia {
	var i *inertia.Inertia

	if cfg.FS != nil {
		i = inertia.NewWithFS(cfg.URL, cfg.RootTemplate, cfg.Version, cfg.FS)
	} else if cfg.Dev {
		templatePath := cfg.RootTemplateDev
		if templatePath == "" {
			templatePath = cfg.RootTemplate
		}
		i = inertia.New(cfg.URL, templatePath, cfg.Version)
	} else {
		i = inertia.New(cfg.URL, cfg.RootTemplate, cfg.Version)
	}

	manifestPath := cfg.ManifestPath
	if manifestPath == "" {
		manifestPath = "public/build/manifest.json"
	}

	viteLoader := vite.NewLoader(manifestPath, cfg.Dev)
	if cfg.ManifestFS != nil {
		viteLoader = vite.NewLoaderWithFS(manifestPath, cfg.Dev, cfg.ManifestFS)
	}

	i.ShareFunc("viteAsset", func(entry string) string {
		path, err := viteLoader.Asset(entry)
		if err != nil {
			if cfg.Dev {
				return "http://localhost:5173/" + entry
			}
			return ""
		}
		return path
	})

	i.ShareFunc("viteCSS", func(entry string) []string {
		paths, err := viteLoader.CSSAssets(entry)
		if err != nil {
			return nil
		}
		return paths
	})

	i.ShareFunc("marshal", func(v any) string {
		data, _ := json.Marshal(v)
		return string(data)
	})

	return i
}
