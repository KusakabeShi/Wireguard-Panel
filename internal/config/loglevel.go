package config

import (
	"encoding/json"
	"fmt"
)

type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelInfo
	LogLevelVerbose
)

var logLevelNames = map[LogLevel]string{
	LogLevelError:   "error",
	LogLevelInfo:    "info",
	LogLevelVerbose: "verbose",
}

var logLevelValues = map[string]LogLevel{
	"error":   LogLevelError,
	"info":    LogLevelInfo,
	"verbose": LogLevelVerbose,
}

func (l LogLevel) MarshalJSON() ([]byte, error) {
	name, ok := logLevelNames[l]
	if !ok {
		name = "unknown"
	}
	return json.Marshal(name)
}
func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if val, ok := logLevelValues[s]; ok {
		*l = val
		return nil
	}
	return fmt.Errorf("invalid log level: %s", s)
}

func (l LogLevel) String() string {
	name, ok := logLevelNames[l]
	if !ok {
		return "unknown"
	}
	return name
}
