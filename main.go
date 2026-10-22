package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/marcopiovanello/yt-dlp-web-ui/v4/server"
	"github.com/marcopiovanello/yt-dlp-web-ui/v4/server/config"
	"github.com/marcopiovanello/yt-dlp-web-ui/v4/server/openid"

	"github.com/spf13/viper"
)

//go:embed frontend/dist/index.html
//go:embed frontend/dist/assets/*
var frontend embed.FS

//go:embed openapi/*
var swagger embed.FS

func main() {
	var (
		configFile       string
		flagPort         int
		flagHost         string
		flagQueueSize    int
		flagDownloadPath string
		flagDriverPath   string
		flagLocalDB      string
		flagRequireAuth  bool
		flagUsername     string
		flagPassword     string
	)
	flag.StringVar(&configFile, "conf", "./config.yml", "Config file path")
	flag.IntVar(&flagPort, "port", 0, "Port where server will listen at")
	flag.StringVar(&flagHost, "host", "", "Host where server will listen at")
	flag.IntVar(&flagQueueSize, "qs", 0, "Queue size (concurrent downloads)")
	flag.StringVar(&flagDownloadPath, "out", "", "Where files will be saved")
	flag.StringVar(&flagDriverPath, "driver", "", "yt-dlp executable path")
	flag.StringVar(&flagLocalDB, "db", "", "local database path")
	flag.BoolVar(&flagRequireAuth, "auth", false, "Enable RPC authentication")
	flag.StringVar(&flagUsername, "user", "", "Username required for auth")
	flag.StringVar(&flagPassword, "pass", "", "Password required for auth")
	flag.Parse()

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 3033)
	defaultQueue := runtime.NumCPU()
	if defaultQueue < 2 {
		defaultQueue = 2
	}
	v.SetDefault("server.queue_size", defaultQueue)
	v.SetDefault("paths.download_path", ".")
	v.SetDefault("paths.downloader_path", "yt-dlp")
	v.SetDefault("paths.local_database_path", ".")
	v.SetDefault("logging.log_path", "yt-dlp-webui.log")
	v.SetDefault("logging.enable_file_logging", false)
	v.SetDefault("authentication.require_auth", false)

	// Env binding
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		slog.Debug("using defaults or config read failed", "error", err)
	}

	// Compatibility aliases for V3 flat config
	aliases := map[string]string{
		"port":                "server.port",
		"host":                "server.host",
		"queue_size":          "server.queue_size",
		"downloadPath":        "paths.download_path",
		"download_path":       "paths.download_path",
		"downloaderPath":      "paths.downloader_path",
		"downloader_path":     "paths.downloader_path",
		"local_database_path": "paths.local_database_path",
		"require_auth":        "authentication.require_auth",
		"username":            "authentication.username",
		"password":            "authentication.password_hash",
		"password_hash":       "authentication.password_hash",
		"log_path":            "logging.log_path",
		"enable_file_logging": "logging.enable_file_logging",
		"frontend_path":       "frontend.frontend_path",
	}

	for flatKey, nestedKey := range aliases {
		if v.InConfig(flatKey) && !v.InConfig(nestedKey) {
			v.Set(nestedKey, v.Get(flatKey))
		}
	}

	// Apply CLI flags if explicitly provided
	if flagPort > 0 {
		v.Set("server.port", flagPort)
	}
	if flagHost != "" {
		v.Set("server.host", flagHost)
	}
	if flagQueueSize > 0 {
		v.Set("server.queue_size", flagQueueSize)
	}
	if flagDownloadPath != "" {
		v.Set("paths.download_path", flagDownloadPath)
	}
	if flagDriverPath != "" {
		v.Set("paths.downloader_path", flagDriverPath)
	}
	if flagLocalDB != "" {
		v.Set("paths.local_database_path", flagLocalDB)
	}
	if flagRequireAuth {
		v.Set("authentication.require_auth", true)
	}
	if flagUsername != "" {
		v.Set("authentication.username", flagUsername)
	}
	if flagPassword != "" {
		v.Set("authentication.password_hash", flagPassword)
	}

	cfg := config.Instance()
	if err := v.Unmarshal(&cfg); err != nil {
		slog.Error("failed to load config", "error", err)
	}

	// Store resolved config file path
	if abs, err := filepath.Abs(configFile); err == nil {
		cfg.SetPath(abs)
	} else {
		cfg.SetPath(configFile)
	}

	// Ensure QueueSize is at least 1
	if cfg.Server.QueueSize <= 0 {
		cfg.Server.QueueSize = 2
	}

	var appFS fs.FS
	if fp := v.GetString("frontend_path"); fp != "" {
		appFS = os.DirFS(fp)
	} else {
		sub, err := fs.Sub(frontend, "frontend/dist")
		if err != nil {
			slog.Error("failed to load embedded frontend", "error", err)
			os.Exit(1)
		}
		appFS = sub
	}

	openid.Configure()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("starting server",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"queue_size", cfg.Server.QueueSize,
		"downloader", cfg.Paths.DownloaderPath,
		"download_dir", cfg.Paths.DownloadPath,
	)

	if err := server.Run(ctx, &server.RunConfig{
		App:     appFS,
		Swagger: swagger,
	}); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("server exited cleanly")
}
