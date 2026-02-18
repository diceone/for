package logger

import (
	"io"
	"log/slog"
	"os"
)

// L is the global structured logger. It is initialised to stdout by default.
var L *slog.Logger

func init() {
	L = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// Init configures the global logger. If logFile is non-empty the output is
// written to both stdout and the file. Returns a cleanup function that must
// be deferred by the caller.
func Init(logFile string) (func(), error) {
	writers := []io.Writer{os.Stdout}
	cleanup := func() {}

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, f)
		cleanup = func() { f.Close() }
	}

	w := io.MultiWriter(writers...)
	L = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(L)
	return cleanup, nil
}
