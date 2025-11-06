package logger

import (
    "encoding/json"
    "fmt"
    "time"
)

type Logger struct{}

func New() *Logger { return &Logger{} }

func (l *Logger) log(level, msg string, fields map[string]interface{}) {
    if fields == nil { fields = map[string]interface{}{} }
    fields["level"] = level
    fields["msg"] = msg
    fields["ts"] = time.Now().Format(time.RFC3339Nano)
    b, _ := json.Marshal(fields)
    fmt.Println(string(b))
}

func (l *Logger) Info(msg string, fields map[string]interface{}) { l.log("info", msg, fields) }
func (l *Logger) Error(msg string, fields map[string]interface{}) { l.log("error", msg, fields) }