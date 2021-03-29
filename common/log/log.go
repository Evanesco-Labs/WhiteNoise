package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	Blue   = "0;34"
	Red    = "0;31"
	Green  = "0;32"
	Yellow = "0;33"
	Cyan   = "0;36"
	Pink   = "1;35"
)

func Color(code, msg string) string {
	return fmt.Sprintf("\033[%sm%s\033[m", code, msg)
}

var LogLevel = InfoLog
var ProcName string
var ModuleLevel map[string]int

const (
	TraceLog = iota
	DebugLog
	InfoLog
	WarnLog
	ErrorLog
	FatalLog
	MaxLevelLog
)

var (
	levels = map[int]string{
		DebugLog: Color(Green, "[DEBUG]"),
		InfoLog:  Color(Cyan, "[INFO ]"),
		WarnLog:  Color(Yellow, "[WARN ]"),
		ErrorLog: Color(Red, "[ERROR]"),
		FatalLog: Color(Pink, "[FATAL]"),
		TraceLog: Color(Blue, "[TRACE]"),
	}
	Stdout = os.Stdout
)

const (
	NAME_PREFIX          = "LEVEL"
	CALL_DEPTH           = 2
	DEFAULT_MAX_LOG_SIZE = 20
	BYTE_TO_MB           = 1024 * 1024
	PATH                 = "./Log/"
)

func SetProcName(name string) {
	ProcName = name
}

func SetModuleLevel(name string, level int) {
	ModuleLevel[name] = level
}

func GetGID() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
func GetPkg() string {
	pc := make([]uintptr, 10)
	runtime.Callers(3, pc)

	f := runtime.FuncForPC(pc[0])

	nameFull := f.Name()

	var pkgName string
	if ProcName == "" || !strings.Contains(nameFull, ProcName) {
		nameEnd := filepath.Base(nameFull)
		pkgName = strings.Split(nameEnd, ".")[0]
	} else {
		nameEnd := nameFull[strings.LastIndex(nameFull, ProcName)+len(ProcName)+1:]
		pkgName = strings.Split(nameEnd, ".")[0]
	}

	return Color(Cyan, "["+pkgName+"]")
}

var Log *Logger

func init() {
	//Default print to console
	InitLog(LogLevel, PATH, Stdout)
}

func LevelName(level int) string {
	if name, ok := levels[level]; ok {
		return name
	}
	return NAME_PREFIX + strconv.Itoa(level)
}

func NameLevel(name string) int {
	for k, v := range levels {
		if v == name {
			return k
		}
	}
	var level int
	if strings.HasPrefix(name, NAME_PREFIX) {
		level, _ = strconv.Atoi(name[len(NAME_PREFIX):])
	}
	return level
}

type Logger struct {
	level   int
	logger  *log.Logger
	logFile *os.File
	ignore  []string
}

func New(out io.Writer, prefix string, flag, level int, file *os.File) *Logger {
	return &Logger{
		level:   level,
		logger:  log.New(out, prefix, flag),
		logFile: file,
		ignore:  make([]string, 0),
	}
}

func (l *Logger) SetDebugLevel(level int) error {
	if level > MaxLevelLog || level < 0 {
		return errors.New("Invalid Debug Level")
	}

	l.level = level
	return nil
}

func (l *Logger) Output(level int, a ...interface{}) error {
	gid := GetGID()
	gidStr := strconv.FormatUint(gid, 10)
	if level <= DebugLog {
		a = append([]interface{}{LevelName(level), "GID",
			gidStr + ","}, a...)
		for k, v := range ModuleLevel {
			if strings.Contains(a[4].(string), k) {
				if level >= v {
					return l.logger.Output(CALL_DEPTH, fmt.Sprintln(a...))
				}
				return nil
			}
		}
	} else {
		pkgName := GetPkg()
		a = append([]interface{}{LevelName(level), "GID",
			gidStr + ", " + pkgName}, a...)
		for k, v := range ModuleLevel {
			if strings.Contains(pkgName, k) {
				if level >= v {
					return l.logger.Output(CALL_DEPTH, fmt.Sprintln(a...))
				}
				return nil
			}
		}

	}
	if level >= l.level {
		//fmt.Printf("level >= l.level\n")
		return l.logger.Output(CALL_DEPTH, fmt.Sprintln(a...))
	}
	return nil
}

func (l *Logger) Outputf(level int, format string, a ...interface{}) error {
	gid := GetGID()
	if level <= DebugLog {
		a = append([]interface{}{LevelName(level), "GID",
			gid}, a...)
		for k, v := range ModuleLevel {
			if strings.Contains(a[4].(string), k) {
				if level >= v {
					return l.logger.Output(CALL_DEPTH, fmt.Sprintf("%s %s %d, "+format+"\n", a...))
				}
				return nil
			}
		}
		if level >= l.level {
			return l.logger.Output(CALL_DEPTH, fmt.Sprintf("%s %s %d, "+format+"\n", a...))
		}
	} else {
		pkgName := GetPkg()
		a = append([]interface{}{LevelName(level), "GID",
			gid, pkgName}, a...)
		for k, v := range ModuleLevel {
			if strings.Contains(pkgName, k) {
				if level >= v {
					return l.logger.Output(CALL_DEPTH, fmt.Sprintf("%s %s %d, %s "+format+"\n", a...))
				}
			}
			return nil
		}
		if level >= l.level {
			return l.logger.Output(CALL_DEPTH, fmt.Sprintf("%s %s %d, %s "+format+"\n", a...))
		}
	}
	return nil
}

func (l *Logger) Trace(a ...interface{}) {
	l.Output(TraceLog, a...)
}

func (l *Logger) Tracef(format string, a ...interface{}) {
	l.Outputf(TraceLog, format, a...)
}

func (l *Logger) Debug(a ...interface{}) {
	l.Output(DebugLog, a...)
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.Outputf(DebugLog, format, a...)
}

func (l *Logger) Info(a ...interface{}) {
	l.Output(InfoLog, a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.Outputf(InfoLog, format, a...)
}

func (l *Logger) Warn(a ...interface{}) {
	l.Output(WarnLog, a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.Outputf(WarnLog, format, a...)
}

func (l *Logger) Error(a ...interface{}) {
	l.Output(ErrorLog, a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Outputf(ErrorLog, format, a...)
}

func (l *Logger) Fatal(a ...interface{}) {
	l.Output(FatalLog, a...)
}

func (l *Logger) Fatalf(format string, a ...interface{}) {
	l.Outputf(FatalLog, format, a...)
}

func AddIgnore(name string) {
	if len(name) == 0 {
		return
	}
	Log.ignore = append(Log.ignore, name)
}

func CleanIgnore() {
	Log.ignore = Log.ignore[:0]
}

func Trace(a ...interface{}) {
	if TraceLog < Log.level && len(ModuleLevel) == 0 {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")
	var pkgName string
	if ProcName == "" || !strings.Contains(nameFull, ProcName) {
		nameEnd := filepath.Base(nameFull)
		pkgName = strings.Split(nameEnd, ".")[0]
	} else {
		nameEnd := nameFull[strings.LastIndex(nameFull, ProcName)+len(ProcName)+1:]
		pkgName = strings.Split(nameEnd, ".")[0]
	}

	a = append([]interface{}{Color(Cyan, "["+pkgName+"]"), funcName + "()", fileName + ":" + strconv.Itoa(line)}, a...)

	Log.Trace(a...)
}

func Tracef(format string, a ...interface{}) {
	if TraceLog < Log.level && len(ModuleLevel) == 0 {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	nameFull := f.Name()
	nameEnd := filepath.Ext(nameFull)
	funcName := strings.TrimPrefix(nameEnd, ".")
	var pkgName string
	if ProcName == "" || !strings.Contains(nameFull, ProcName) {
		nameEnd := filepath.Base(nameFull)
		pkgName = strings.Split(nameEnd, ".")[0]
	} else {
		nameEnd := nameFull[strings.LastIndex(nameFull, ProcName)+len(ProcName)+1:]
		pkgName = strings.Split(nameEnd, ".")[0]
	}
	a = append([]interface{}{Color(Cyan, "["+pkgName+"]"), funcName, fileName, line}, a...)

	Log.Tracef("%s %s() %s:%d "+format, a...)
}

func Debug(a ...interface{}) {
	if DebugLog < Log.level && len(ModuleLevel) == 0 {
		return
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)

	for _, ig := range Log.ignore {
		if strings.Index(file, ig) != -1 {
			return
		}
	}
	nameFull := f.Name()
	var pkgName string
	if ProcName == "" || !strings.Contains(nameFull, ProcName) {
		nameEnd := filepath.Base(nameFull)
		pkgName = strings.Split(nameEnd, ".")[0]
	} else {
		nameEnd := nameFull[strings.LastIndex(nameFull, ProcName)+len(ProcName)+1:]
		pkgName = strings.Split(nameEnd, ".")[0]
	}
	a = append([]interface{}{Color(Cyan, "["+pkgName+"]"), f.Name(), fileName + ":" + strconv.Itoa(line)}, a...)

	Log.Debug(a...)
}

func Debugf(format string, a ...interface{}) {
	if DebugLog < Log.level && len(ModuleLevel) == 0 {
		return
	}
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fileName := filepath.Base(file)
	for _, ig := range Log.ignore {
		if strings.Index(file, ig) != -1 {
			return
		}
	}
	nameFull := f.Name()
	var pkgName string
	if ProcName == "" || !strings.Contains(nameFull, ProcName) {
		nameEnd := filepath.Base(nameFull)
		pkgName = strings.Split(nameEnd, ".")[0]
	} else {
		nameEnd := nameFull[strings.LastIndex(nameFull, ProcName)+len(ProcName)+1:]
		pkgName = strings.Split(nameEnd, ".")[0]
	}
	a = append([]interface{}{Color(Cyan, "["+pkgName+"]"), f.Name(), fileName, line}, a...)

	Log.Debugf("%s %s %s:%d "+format, a...)
}

func Info(a ...interface{}) {
	Log.Info(a...)
}

func Warn(a ...interface{}) {
	Log.Warn(a...)
}

func Error(a ...interface{}) {
	Log.Error(a...)
}

func Fatal(a ...interface{}) {
	Log.Fatal(a...)
}

func Infof(format string, a ...interface{}) {
	Log.Infof(format, a...)
}

func Warnf(format string, a ...interface{}) {
	Log.Warnf(format, a...)
}

func Errorf(format string, a ...interface{}) {
	Log.Errorf(format, a...)
}

func Fatalf(format string, a ...interface{}) {
	Log.Fatalf(format, a...)
}

func FileOpen(path string) (*os.File, error) {
	if fi, err := os.Stat(path); err == nil {
		if !fi.IsDir() {
			return nil, fmt.Errorf("open %s: not a directory", path)
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0766); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	var currenttime = time.Now().Format("2006-01-02_15.04.05")

	logfile, err := os.OpenFile(path+currenttime+"_LOG.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return logfile, nil
}

//Init deprecated, use InitLog instead
//func Init(a ...interface{}) {
//	os.Stderr.WriteString("warning: use of deprecated Init. Use InitLog instead\n")
//	InitLog(InfoLog, a...)
//}

func InitLog(logLevel int, a ...interface{}) {
	writers := []io.Writer{}
	var logFile *os.File
	var err error
	if len(a) == 0 {
		writers = append(writers, ioutil.Discard)
	} else {
		for _, o := range a {
			switch o.(type) {
			case string:
				logFile, err = FileOpen(o.(string))
				if err != nil {
					fmt.Println("error: open log file failed")
					os.Exit(1)
				}
				writers = append(writers, logFile)
			case *os.File:
				writers = append(writers, o.(*os.File))
			default:
				fmt.Println("error: invalid log location")
				os.Exit(1)
			}
		}
	}
	fileAndStdoutWrite := io.MultiWriter(writers...)
	Log = New(fileAndStdoutWrite, "", log.Ldate|log.Lmicroseconds, logLevel, logFile)
	ModuleLevel = make(map[string]int)
}

func GetLogFileSize() (int64, error) {
	f, e := Log.logFile.Stat()
	if e != nil {
		return 0, e
	}
	return f.Size(), nil
}

func GetMaxLogChangeInterval(maxLogSize int64) int64 {
	if maxLogSize != 0 {
		return (maxLogSize * BYTE_TO_MB)
	} else {
		return (DEFAULT_MAX_LOG_SIZE * BYTE_TO_MB)
	}
}

func CheckIfNeedNewFile() bool {
	logFileSize, err := GetLogFileSize()
	maxLogFileSize := GetMaxLogChangeInterval(0)
	if err != nil {
		return false
	}
	if logFileSize > maxLogFileSize {
		return true
	} else {
		return false
	}
}

func ClosePrintLog() error {
	var err error
	if Log.logFile != nil {
		err = Log.logFile.Close()
	}
	return err
}
