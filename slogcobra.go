package slogcobra

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chg1f/storageunit"
	"github.com/mitchellh/mapstructure"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

type Config struct {
	Format       string            `json:"format"`
	Level        string            `json:"level"`
	FilePath     string            `json:"file_path"`
	FileCount    int               `json:"file_count"`
	FileSize     storageunit.Bytes `json:"file_size"`
	FileDuration time.Duration     `json:"file_duration"`
	FileCompress bool              `json:"file_compress"`
}

func newHandler(conf *Config) (slog.Handler, error) {
	var (
		out io.Writer
		lvl slog.Level
	)
	switch conf.FilePath {
	case "", "/dev/null":
		out = io.Discard
	case "stdout":
		out = os.Stdout
	case "stderr":
		out = os.Stderr
	default:
		if err := os.MkdirAll(filepath.Dir(conf.FilePath), 0755); err != nil {
			return nil, err
		}
		maxSize := 100
		if conf.FileSize != 0 {
			maxSize = int(conf.FileSize.Megabytes())
		}
		maxAge := 1
		if conf.FileDuration != 0 {
			maxAge = int(conf.FileDuration / (time.Hour * 24))
		}
		out = &lumberjack.Logger{
			Filename:   conf.FilePath,
			MaxBackups: conf.FileCount,
			MaxSize:    maxSize,
			MaxAge:     maxAge,
			Compress:   conf.FileCompress,
			LocalTime:  true,
		}
	}
	switch conf.Format {
	case "json":
		handler := slog.NewJSONHandler(out, &slog.HandlerOptions{
			Level: lvl,
		})
		return handler, nil
	case "text", "":
		handler := slog.NewTextHandler(out, &slog.HandlerOptions{
			Level: lvl,
		})
		return handler, nil
	default:
		return nil, fmt.Errorf("unknown log format: %s", conf.Format)
	}
}

var (
	NewHandler = newHandler
)

func init() {
	cobra.OnInitialize(func() {
		var conf Config
		if err := viper.UnmarshalKey("log", &conf, func(dc *mapstructure.DecoderConfig) {
			dc.TagName = "json"
			dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
				mapstructure.RecursiveStructToMapHookFunc(),
				mapstructure.StringToTimeDurationHookFunc(),
				storageunit.StringToBytesHookFunc(),
			)
		}); err != nil {
			panic(err)
		}
		handler, err := newHandler(&conf)
		if err != nil {
			panic(err)
		}
		slog.SetDefault(slog.New(handler))
	})
	viper.SetDefault("log", &Config{
		Format:       "text",
		Level:        "info",
		FilePath:     "stderr",
		FileCount:    10,
		FileSize:     storageunit.Megabyte * 100,
		FileDuration: time.Hour * 24 * 30,
		FileCompress: true,
	})
}
