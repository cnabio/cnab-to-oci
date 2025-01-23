package remotes

import (
	"context"
	"encoding/json"
	"io"

	"github.com/containerd/log"
	"github.com/sirupsen/logrus"
)

func logPayload(logger *logrus.Entry, payload interface{}) {
	buf, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	logger.Debug(string(buf))
}

func withMutedContext(ctx context.Context) context.Context {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	logger.SetOutput(io.Discard)
	return log.WithLogger(ctx, logrus.NewEntry(logger))
}
