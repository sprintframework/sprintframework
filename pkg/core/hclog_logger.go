/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
)

type hclogAdapter struct {
	log  *zap.Logger
	name string
}

func newHCLogAdapter(log *zap.Logger) hclog.Logger {
	return hclogAdapter{log: log}
}

// Args are alternating key, val pairs
// keys must be strings
// vals can be any type, but display is implementation specific
// Emit a message and key/value pairs at a provided log level
func (t hclogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Debug:
		t.Debug(msg, args...)
	case hclog.Warn:
		t.Warn(msg, args...)
	case hclog.Error:
		t.Error(msg, args...)
	case hclog.DefaultLevel, hclog.Info, hclog.NoLevel, hclog.Off, hclog.Trace:
		t.Info(msg, args...)
	}
}

// Emit a message and key/value pairs at the TRACE level
func (t hclogAdapter) Trace(msg string, args ...interface{}) {
	t.log.Info(msg, toZapFields(args...)...)
}

// Emit a message and key/value pairs at the DEBUG level
func (t hclogAdapter) Debug(msg string, args ...interface{}) {
	t.log.Debug(msg, toZapFields(args...)...)
}

// Emit a message and key/value pairs at the INFO level
func (t hclogAdapter) Info(msg string, args ...interface{}) {
	t.log.Info(msg, toZapFields(args...)...)
}

// Emit a message and key/value pairs at the WARN level
func (t hclogAdapter) Warn(msg string, args ...interface{}) {
	t.log.Warn(msg, toZapFields(args...)...)
}

// Emit a message and key/value pairs at the ERROR level
func (t hclogAdapter) Error(msg string, args ...interface{}) {
	t.log.Error(msg, toZapFields(args...)...)
}

// Indicate if TRACE logs would be emitted. This and the other Is* guards
// are used to elide expensive logging code based on the current level.
func (t hclogAdapter) IsTrace() bool { return false }

// Indicate if DEBUG logs would be emitted. This and the other Is* guards
func (t hclogAdapter) IsDebug() bool { return false }

// Indicate if INFO logs would be emitted. This and the other Is* guards
func (t hclogAdapter) IsInfo() bool { return false }

// Indicate if WARN logs would be emitted. This and the other Is* guards
func (t hclogAdapter) IsWarn() bool { return false }

// Indicate if ERROR logs would be emitted. This and the other Is* guards
func (t hclogAdapter) IsError() bool { return false }

// ImpliedArgs returns With key/value pairs
func (t hclogAdapter) ImpliedArgs() []interface{} { return nil }

// Creates a sublogger that will always have the given key/value pairs
func (t hclogAdapter) With(args ...interface{}) hclog.Logger {
	return hclogAdapter{log: t.log.With(toZapFields(args...)...)}
}

// Returns the Name of the logger
func (t hclogAdapter) Name() string {
	return t.name
}

// Create a logger that will prepend the name string on the front of all messages.
// If the logger already has a name, the new value will be appended to the current
// name. That way, a major subsystem can use this to decorate all it's own logs
// without losing context.
func (t hclogAdapter) Named(name string) hclog.Logger {
	return &hclogAdapter{log: t.log.Named(name), name: name}
}

// Create a logger that will prepend the name string on the front of all messages.
// This sets the name of the logger to the value directly, unlike Named which honor
// the current name as well.
func (t hclogAdapter) ResetNamed(name string) hclog.Logger {
	return &hclogAdapter{log: t.log.Named(name), name: name}
}

// Updates the level. This should affect all related loggers as well,
// unless they were created with IndependentLevels. If an
// implementation cannot update the level on the fly, it should no-op.
func (t hclogAdapter) SetLevel(level hclog.Level) {
}

// Returns the current level
func (t hclogAdapter) GetLevel() hclog.Level {
	return toHCLevel(t.log.Level())
}

// Return a value that conforms to the stdlib log.Logger interface
func (t hclogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	if opts.ForceLevel == hclog.NoLevel {
		return zap.NewStdLog(t.log)
	}
	if stdLogger, err := zap.NewStdLogAt(t.log, toZapLevel(opts.ForceLevel)); err != nil {
		return zap.NewStdLog(t.log)
	} else {
		return stdLogger
	}
}

// Return a value that conforms to io.Writer, which can be passed into log.SetOutput()
func (t hclogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return zap.NewStdLog(t.log).Writer()
}

func toZapLevel(level hclog.Level) zapcore.Level {
	switch level {
	case hclog.Trace, hclog.Debug:
		return zapcore.DebugLevel
	case hclog.Info:
		return zapcore.InfoLevel
	case hclog.Warn:
		return zapcore.WarnLevel
	case hclog.Error:
		return zapcore.ErrorLevel
	case hclog.Off:
		return zapcore.FatalLevel
	}
	return zapcore.InfoLevel // default level
}

func toHCLevel(level zapcore.Level) hclog.Level {
	switch level {
	case zapcore.DebugLevel:
		return hclog.Debug
	case zapcore.InfoLevel:
		return hclog.Info
	case zapcore.WarnLevel:
		return hclog.Warn
	case zapcore.ErrorLevel:
		return hclog.Error
	case zapcore.DPanicLevel:
		return hclog.Error
	case zapcore.PanicLevel:
		return hclog.Error
	case zapcore.FatalLevel:
		return hclog.Off
	}
	return hclog.NoLevel // default level
}

func toZapFields(args ...interface{}) []zapcore.Field {
	var fields []zapcore.Field
	for i := len(args); i > 0; i -= 2 {
		left := i - 2
		if left < 0 {
			left = 0
		}

		items := args[left:i]

		switch l := len(items); l {
		case 2:
			k, ok := items[0].(string)
			if ok {
				fields = append(fields, zap.Any(k, items[1]))
			} else {
				fields = append(fields, zap.Any(fmt.Sprintf("arg%d", i-1), items[1]))
				fields = append(fields, zap.Any(fmt.Sprintf("arg%d", left), items[0]))
			}
		case 1:
			fields = append(fields, zap.Any(fmt.Sprintf("arg%d", left), items[0]))
		}
	}

	return fields
}

