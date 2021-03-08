package log

import (
	"errors"
	"regexp"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.OutputPaths = []string{"stdout"}
	zapLogger, _ := logConfig.Build()
	zapCore = &zapcoreWrapper{Core: zapLogger.Core()}
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
var zapCore *zapcoreWrapper

var levelResolver = func(name string) zapcore.Level {
	return zapcore.InfoLevel
}

func SetupLogging(logger zapcore.Core, resolver func(string) zapcore.Level, opts ...zap.Option) {
	// TODO : cleanup
	//if z, ok := zapCore.(*zapcoreWrapper);ok {
	//	z.Core = logger
	//}

	//zapCore = logger

	zapCore.Core = logger
	levelResolver = resolver
	zapOptions = opts

	loggerMutex.Lock()
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
	zapcore.Core
}
