package updatelog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Keep verbose update diagnostics while the standalone updater stabilizes.
const Filename = "update.log"

func Path(dataDir string) string {
	return filepath.Join(dataDir, "logs", Filename)
}

func NewSessionID() string {
	return fmt.Sprintf("%s-p%d", time.Now().UTC().Format("20060102T150405.000000000Z"), os.Getpid())
}

func Open(path string, sessionID string, component string) (*log.Logger, io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return log.New(file, logPrefix(sessionID, component), log.LstdFlags|log.Lmicroseconds|log.LUTC), file, nil
}

func Discard(sessionID string, component string) *log.Logger {
	return log.New(io.Discard, logPrefix(sessionID, component), log.LstdFlags|log.Lmicroseconds|log.LUTC)
}

func logPrefix(sessionID string, component string) string {
	return fmt.Sprintf("[%s] [%s] ", sessionID, component)
}
