package audit

import (
	"io"
	"log/slog"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
)

type Logger struct {
	logger *slog.Logger
}

func New(w io.Writer) *Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				a.Key = "timestamp"
			}
			return a
		},
	})
	return &Logger{logger: slog.New(handler)}
}

func (l *Logger) Record(requestID, workload, destination, method string, decision policy.Effect, reason string) {
	l.logger.Info("egress_decision",
		"request_id", requestID,
		"workload", workload,
		"destination", destination,
		"method", method,
		"decision", decision,
		"reason", reason,
	)
}
