package log

import (
	"go.uber.org/zap/zapcore"
)

// Created new interface without LevelEnabler because interface overlapping didn't work yet on 1.13
type AlmostCore interface {
	With([]zapcore.Field) zapcore.Core
	Check(zapcore.Entry, *zapcore.CheckedEntry) *zapcore.CheckedEntry
	Write(zapcore.Entry, []zapcore.Field) error
	Sync() error
}

func WrapCore(core zapcore.Core, level zapcore.LevelEnabler) zapcore.Core {
	return &coreWrapper{
		LevelEnabler: level,
		AlmostCore:   core,
	}
}

type coreWrapper struct {
	zapcore.LevelEnabler
	AlmostCore
}

func (c *coreWrapper) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}
