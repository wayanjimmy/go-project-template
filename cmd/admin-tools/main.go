package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"testing/fstest"

	assets "go-project-template/assets"
	"go-project-template/buildinfo"
	"go-project-template/config"
	"go-project-template/logger"
	"go-project-template/publisher"
	"go-project-template/repository"
	"go-project-template/requestid"
	"go-project-template/service"
	"go-project-template/setup"
	"go-project-template/vite"
	"go-project-template/web"
	"go-project-template/worker"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/go-chi/chi/v5"
)

const rootTemplateHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Admin Tools</title>
</head>
<body>
  <div id="app" data-page="{{ marshal .page }}"></div>
  <script type="module" src="{{ viteAsset "cmd/admin-tools/resources/js/app.tsx" }}"></script>
</body>
</html>
`

var rootTemplateFS = fstest.MapFS{
	"resources/views/app.gohtml": &fstest.MapFile{Data: []byte(rootTemplateHTML)},
}

func main() {
	correlationIDFn := requestid.FromContext
	ctx := context.Background()

	buildinfo.Service = "APP-ADMIN-TOOLS"
	log := logger.New(os.Stdout, logger.LevelInfo, buildinfo.Service, correlationIDFn)

	logger.BuildInfo(ctx, log)

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "error", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	cfg := &config.Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ServerPort:  envOrDefault("ADMIN_TOOLS_PORT", "8081"),

		SecretType:             envOrDefault("SECRET_DRIVER", "FILESYSTEM"),
		SecretRoot:             envOrDefault("SECRET_ROOT", ".secrets"),
		SecretEncryptionParent: envOrDefault("SECRET_ENCRYPTION_PARENT", "local"),
		SecretEncryptionName:   envOrDefault("SECRET_ENCRYPTION_NAME", "user-data"),
	}

	env, err := setup.Setup(ctx, log, cfg)
	if err != nil {
		return fmt.Errorf("setup.Setup: %w", err)
	}
	defer env.Close(ctx)

	db := env.Database()
	if db == nil {
		return fmt.Errorf("database not configured")
	}

	encryptor := env.DataEncryptor()
	if encryptor == nil {
		return fmt.Errorf("data encryptor not configured")
	}

	repos := repository.NewPostgresRepositories(db, encryptor, log)

	watermillLogger := logger.NewWatermillAdapter(log)
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, watermillLogger)

	decoratedPub, err := publisher.CorrelationIDDecorator()(pubSub)
	if err != nil {
		return fmt.Errorf("failed to decorate publisher: %w", err)
	}
	eventPub := publisher.NewWatermillPublisher(decoratedPub)

	userService := service.NewUserService(repos.UserRepo, eventPub, log)
	searchService := service.NewSearchService(repos.UserSearcher)

	watermillRouter, err := worker.NewRouter(pubSub, repos.UserIndexer, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create worker router: %w", err)
	}
	defer watermillRouter.Close()

	go func() {
		if err := watermillRouter.Run(ctx); err != nil {
			log.Error(ctx, "worker", "error", err.Error())
		}
	}()

	devMode := envOrDefault("APP_ENV", "development") != "production"
	if !devMode {
		if _, err := vite.LoadFromFS(assets.PublicBuildFS, "public/build/manifest.json"); err != nil {
			return fmt.Errorf("embedded frontend assets missing (run `task ui:build` before building binary): %w", err)
		}
	}

	i := web.NewInertia(web.InertiaConfig{
		Dev:             devMode,
		RootTemplate:    "resources/views/app.gohtml",
		RootTemplateDev: "cmd/admin-tools/resources/views/app.gohtml",
		URL:             "http://localhost:" + cfg.ServerPort,
		Version:         buildinfo.Version,
		FS:              rootTemplateFS,
		ManifestPath:    "public/build/manifest.json",
		ManifestFS:      assets.PublicBuildFS,
	})

	r := chi.NewRouter()
	r.Use(i.Middleware)

	usersPage := web.NewUsersPageHandler(i, userService, searchService, log)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/users", http.StatusFound)
	})
	r.Get("/users", usersPage.Index)

	buildFS, err := fs.Sub(assets.PublicBuildFS, "public/build")
	if err != nil {
		return fmt.Errorf("sub fs public/build: %w", err)
	}
	r.Handle("/build/*", http.StripPrefix("/build", http.FileServer(http.FS(buildFS))))

	api := &http.Server{Addr: ":" + cfg.ServerPort, Handler: r}
	log.Info(ctx, "startup", "status", "admin tools started", "addr", api.Addr)
	if err := api.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
