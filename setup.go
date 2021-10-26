package log

import (
	"errors"
	"regexp"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.OutputPaths = []string{"stdout"}
	zapLogger, _ := logConfig.Build()
	zapCore.setCore(zapLogger.Core())
}

// ErrNoSuchLogger is returned when the util pkg is asked for a non existent logger
var ErrNoSuchLogger = errors.New("Error: No such logger")

// loggers is the set of loggers in the system
var loggerMutex sync.RWMutex
var loggers = make(map[string]*zap.SugaredLogger)
var levels = make(map[string]zap.AtomicLevel)

// Params for creating new loggers
var zapOptions []zap.Option

//var zapCore zapcore.Core
var zapCore = &zapcoreWrapper{}

var levelResolver = func(name string) zapcore.Level {
	return zapcore.InfoLevel
}

func SetupLogging(logger zapcore.Core, resolver func(string) zapcore.Level, opts ...zap.Option) {
	// TODO : cleanup
	//if z, ok := zapCore.(*zapcoreWrapper);ok {
	//	z.Core = logger
	//}

	//zapCore = logger

	loggerMutex.Lock()
	zapCore.setCore(logger)
	levelResolver = resolver
	zapOptions = opts

	for name := range loggers {
		levels[name].SetLevel(resolver(name))
	}
	loggerMutex.Unlock()
}

// SetDebugLogging calls SetAllLoggers with logging.DEBUG
func SetDebugLogging() {
	SetAllLoggers(LevelDebug)
}

// SetAllLoggers changes the logging level of all loggers to lvl
func SetAllLoggers(lvl LogLevel) {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	for _, l := range levels {
		l.SetLevel(zapcore.Level(lvl))
	}
}

// SetLogLevel changes the log level of a specific subsystem
// name=="*" changes all subsystems
func SetLogLevel(name, level string) error {
	lvl, err := LevelFromString(level)
	if err != nil {
		return err
	}

	// wildcard, change all
	if name == "*" {
		SetAllLoggers(lvl)
		return nil
	}

	loggerMutex.RLock()
	defer loggerMutex.RUnlock()

	// Check if we have a logger by that name
	if _, ok := levels[name]; !ok {
		return ErrNoSuchLogger
	}

	levels[name].SetLevel(zapcore.Level(lvl))

	return nil
}

// SetLogLevelRegex sets all loggers to level `l` that match expression `e`.
// An error is returned if `e` fails to compile.
func SetLogLevelRegex(e, l string) error {
	lvl, err := LevelFromString(l)
	if err != nil {
		return err
	}

	rem, err := regexp.Compile(e)
	if err != nil {
		return err
	}

	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	for name := range loggers {
		if rem.MatchString(name) {
			levels[name].SetLevel(zapcore.Level(lvl))
		}
	}
	return nil
}

// GetSubsystems returns a slice containing the
// names of the current loggers
func GetSubsystems() []string {
	loggerMutex.RLock()
	defer loggerMutex.RUnlock()
	subs := make([]string, 0, len(loggers))

	for k := range loggers {
		subs = append(subs, k)
	}
	return subs
}

func getLogger(name string) *zap.SugaredLogger {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	log, ok := loggers[name]
	if !ok {
		lvl := zap.NewAtomicLevelAt(levelResolver(name))
		levels[name] = lvl

		newCore := WrapCore(zapCore, lvl)
		newLogger := zap.New(newCore, zapOptions...)

		log = newLogger.Named(name).Sugar()
		loggers[name] = log
	}

	return log
}

type zapcoreWrapper struct {
	core atomic.Value
}

func (z *zapcoreWrapper) setCore(core zapcore.Core) {
	z.core.Store(core)
}

func (z *zapcoreWrapper) getCore() zapcore.Core {
	return z.core.Load().(zapcore.Core)
}

func (z *zapcoreWrapper) Enabled(level zapcore.Level) bool {
	return z.getCore().Enabled(level)
}

func (z *zapcoreWrapper) With(fields []zapcore.Field) zapcore.Core {
	return z.getCore().With(fields)
}

func (z *zapcoreWrapper) Check(entry zapcore.Entry, entry2 *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return z.getCore().Check(entry, entry2)
}

func (z *zapcoreWrapper) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return z.getCore().Write(entry, fields)
}

func (z *zapcoreWrapper) Sync() error {
	return z.getCore().Sync()
}
