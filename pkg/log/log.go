package log

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
)

type ParamNew struct {
	Out     io.Writer
	Color   bool
	Level   string
	Version string
}

// func New(param *ParamNew) (*slog.Logger, error) {
// 	opts := []clog.Option{
// 		clog.WithWriter(param.Out),
// 		clog.WithColor(param.Color),
// 		clog.WithSource(true),
// 	}
// 	if param.Level != "" {
// 		level, err := parseLevel(param.Level)
// 		if err != nil {
// 			return nil, fmt.Errorf("parse a log level: %w", err)
// 		}
// 		opts = append(opts, clog.WithLevel(level))
// 	}
// 	handler := clog.New(opts...)
// 	return slog.New(handler).With(
// 		slog.String("program", "lintnet"),
// 		slog.String("lintnet_version", param.Version),
// 		slog.String("env", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)),
// 	), nil
// }

func New(param *ParamNew) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{
		AddSource: true,
	}
	if param.Level != "" {
		level, err := parseLevel(param.Level)
		if err != nil {
			return nil, fmt.Errorf("parse a log level: %w", err)
		}
		opts.Level = level
	}
	handler := slog.NewJSONHandler(param.Out, opts)
	return slog.New(handler).With(
		slog.String("program", "lintnet"),
		slog.String("lintnet_version", param.Version),
		slog.String("env", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)),
	), nil
}

func parseLevel(level string) (slog.Level, error) {
	m := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	if l, ok := m[level]; ok {
		return l, nil
	}
	return slog.LevelError, errors.New("unknown log level")
}
