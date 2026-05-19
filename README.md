[![codecov](https://codecov.io/gh/coding-kelps/prettyslog/graph/badge.svg?token=ZXH1M9P5HG)](https://codecov.io/gh/coding-kelps/prettyslog)

# prettyslog

A minimalist fork of [Dustin Moris' slog handler](https://github.com/dusted-go/logging).

## Examples

```go
import (
	"log/slog"
	"os"

	"github.com/coding-kelps/prettyslog"
)

func main() {
	// Create a pretty handler
	handler := prettyslog.NewHandler(&slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(handler)
	logger.Info("Application started", "env", "dev")
}
```

## Credits

All credits goes to [Dustin Moris](https://github.com/dustinmoris).
