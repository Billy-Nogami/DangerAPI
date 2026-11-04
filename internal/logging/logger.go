// Package logging настраивает единый формат журналов приложения.
package logging

import (
	"io"
	"log/slog"
)

// New создаёт журнал в текстовом структурированном формате.
func New(writer io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
