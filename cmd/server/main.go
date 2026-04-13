package main

import (
	"log/slog"

	"github.com/fpawel/signature-service/internal"
	"github.com/fpawel/signature-service/internal/logh"
	"github.com/fpawel/signature-service/internal/server"
)

type Args struct {
	Port int `arg:"positional"  help:"Service listen port"`
}

func main() {
	logHandler := logh.NewLogHandler(slog.LevelDebug)
	slog.SetDefault(slog.New(logHandler))

	var args Args
	internal.MustParseArgs(&args)
	internal.ExitErr(server.Run(args.Port))
}
