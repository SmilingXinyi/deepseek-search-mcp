package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	mu       sync.Mutex
	minLevel = LevelInfo
	file     *os.File
	fileSize int64
	maxBytes int64 = 5 * 1024 * 1024
	maxFiles       = 5
	filePath       = ""
)

func Init() {
	minLevel = parseLevel(os.Getenv("MCP_LOG_LEVEL"))
	maxBytes = parseInt64(os.Getenv("MCP_LOG_MAX_BYTES"), 5*1024*1024)
	maxFiles = int(parseInt64(os.Getenv("MCP_LOG_MAX_FILES"), 5))
	if os.Getenv("MCP_LOG_FILE") == "0" {
		return
	}
	dir := os.Getenv("MCP_LOG_DIR")
	if dir == "" {
		dir = "logs"
	}
	_ = os.MkdirAll(dir, 0o755)
	filePath = filepath.Join(dir, "mcp-server.log")
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logger: failed to open log file:", err)
		return
	}
	file = f
	if st, err := f.Stat(); err == nil {
		fileSize = st.Size()
	}
}

func parseLevel(raw string) Level {
	switch strings.ToLower(raw) {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func parseInt64(raw string, fallback int64) int64 {
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

func write(level Level, label, msg string) {
	if level < minLevel {
		return
	}
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().UTC().Format(time.RFC3339Nano), label, msg)
	// Never write to stdout — reserved for MCP stdio protocol.
	_, _ = io.WriteString(os.Stderr, line)

	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	if fileSize+int64(len(line)) > maxBytes {
		rotateLocked()
	}
	n, err := file.WriteString(line)
	if err == nil {
		fileSize += int64(n)
	}
}

func rotateLocked() {
	if file != nil {
		_ = file.Close()
		file = nil
	}
	for i := maxFiles - 1; i >= 1; i-- {
		src := filePath
		if i > 1 {
			src = fmt.Sprintf("%s.%d", filePath, i-1)
		}
		dest := fmt.Sprintf("%s.%d", filePath, i)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dest)
		}
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logger: rotate failed:", err)
		return
	}
	file = f
	fileSize = 0
}

func Debug(msg string) { write(LevelDebug, "DEBUG", msg) }
func Info(msg string)  { write(LevelInfo, "INFO", msg) }
func Warn(msg string)  { write(LevelWarn, "WARN", msg) }
func Error(msg string) { write(LevelError, "ERROR", msg) }
