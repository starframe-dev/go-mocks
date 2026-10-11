package contracts

import (
	"log/slog"

	"github.com/starframe-dev/go-mocks/logger"
)

// AssertLoggerContract compiles at compile time that MemoryHandler implements slog.Handler.
var _ slog.Handler = (*logger.MemoryHandler)(nil)
