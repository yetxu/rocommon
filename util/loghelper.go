package util

import (
	"bytes"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	Trace = iota
	Debug
	Info
	Warning
	Error
	Fatal
)
const Max_File_Bites = 1024 * 1024 * 500 //500M

var logger *log.Logger
var uniLogger *log.Logger

//var (
//	logMu          sync.Mutex
//	path           string = "./log"
//	prefix         string
//	fileName       string
//	uniLogFileName string
//	level          int
//	fileLogging    *os.File
//	fileIndex      int
//	logMaxSize     int64
//)

type LogStInfo struct {
	logMu       sync.Mutex
	path        string
	prefix      string
	fileName    string
	level       int
	fileLogging *os.File
	fileIndex   int
	logMaxSize  int64
}

var commonLogInfo *LogStInfo
var uniLogInfo *LogStInfo

func logFileName(index int, info *LogStInfo) string {
	//format格式説明https://segmentfault.com/q/1010000010976398/a-1020000010982052
	if index == 0 {
		loc := GetLoc()
		strbuf := bytes.NewBufferString(info.path)
		strbuf.WriteString("/")
		strbuf.WriteString(info.prefix)
		strbuf.WriteString(time.Now().In(loc).Format("20060102"))
		strbuf.WriteString(".log")
		return strbuf.String()
		//return fmt.Sprintf("%v/%v%v.log", path, prefix, time.Now().Format("2006010215"))
	} else {
		loc := GetLoc()
		strbuf := bytes.NewBufferString(info.path)
		strbuf.WriteString("/")
		strbuf.WriteString(info.prefix)
		strbuf.WriteString(time.Now().In(loc).Format("20060102"))
		strbuf.WriteString("_")
		strbuf.WriteString(strconv.Itoa(index))
		strbuf.WriteString(".log")
		return strbuf.String()

		//return fmt.Sprintf("%v/%v%v_%v.log", path, prefix, time.Now().Format("2006010215"),index)
	}
}

func pathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	ret := err == nil
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return ret, nil
}

func InitLog(logLevel, maxSize int, pathStr, prefixStr string, uniPathStr string) error {
	if pathStr != "" {
		if ret, err := pathExist(pathStr); !ret {
			if err != nil {
				return err
			}
			err := os.MkdirAll(pathStr, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}
	if uniPathStr != "" {
		if ret, err := pathExist(uniPathStr); !ret {
			if err != nil {
				return err
			}
			err := os.MkdirAll(uniPathStr, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	commonLogInfo = &LogStInfo{}
	commonLogInfo.path = pathStr
	commonLogInfo.prefix = prefixStr
	commonLogInfo.level = logLevel
	commonLogInfo.fileName = logFileName(0, commonLogInfo)
	commonLogInfo.fileIndex = 0
	commonLogInfo.logMaxSize = int64(maxSize) * 1024 * 1024
	if commonLogInfo.logMaxSize <= 0 {
		commonLogInfo.logMaxSize = Max_File_Bites
	}

	uniLogInfo = &LogStInfo{}
	uniLogInfo.path = uniPathStr
	uniLogInfo.prefix = "gamestatistic_" + prefixStr
	uniLogInfo.fileName = logFileName(0, uniLogInfo)
	uniLogInfo.fileIndex = 0
	uniLogInfo.logMaxSize = int64(maxSize) * 1024 * 1024
	if uniLogInfo.logMaxSize <= 0 {
		uniLogInfo.logMaxSize = Max_File_Bites
	}
	//path = pathStr
	//prefix = prefixStr
	//level = logLevel
	//fileName = logFileName(0)
	//fileIndex = 0
	//logMaxSize = int64(maxSize) * 1024 * 1024
	//if logMaxSize <= 0 {
	//	logMaxSize = Max_File_Bites
	//}

	newLogger()
	if uniPathStr != "" {
		newUniLogger()
	}
	return nil
}

func newLogger() {
	var err error = nil
	commonLogInfo.fileLogging, err = os.OpenFile(commonLogInfo.fileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	//fileLogging, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		panic(err)
	}
	logger = log.New(commonLogInfo.fileLogging, "", log.LstdFlags)
}
func newUniLogger() {
	var err error = nil
	uniLogInfo.fileLogging, err = os.OpenFile(uniLogInfo.fileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	//fileLogging, err = os.OpenFile(uniLogFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		panic(err)
	}
	uniLogger = log.New(uniLogInfo.fileLogging, "", log.LUTC)
}

func DebugF(fmt string, v ...interface{}) {
	writeLog(commonLogInfo, Debug, fmt, v...)
}
func InfoF(fmt string, v ...interface{}) {
	writeLog(commonLogInfo, Info, fmt, v...)
}
func WarnF(fmt string, v ...interface{}) {
	writeLog(commonLogInfo, Warning, fmt, v...)
}
func ErrorF(fmt string, v ...interface{}) {
	writeLog(commonLogInfo, Error, fmt, v...)
}
func FatalF(fmt string, v ...interface{}) {
	writeLog(commonLogInfo, Fatal, fmt, v...)
}
func PanicF(fmt string, v ...interface{}) {
	writeLog(commonLogInfo, Fatal, fmt, v...)
	panic(v)
}
func SpecialF(fmt string, v ...interface{}) {
	specialWriteLog(uniLogInfo, fmt, v...)
}

func writeLog(info *LogStInfo, logLevel int, logFmt string, v ...interface{}) {
	if logLevel < info.level {
		return
	}

	info.logMu.Lock()
	newFileName := logFileName(0, info)
	if info.fileName != newFileName {
		if info.fileLogging != nil {
			info.fileLogging.Close()
		}
		info.fileName = newFileName
		info.fileIndex = 0
		newLogger()
	} else {
		for {
			fi, err := info.fileLogging.Stat()
			if err == nil {
				if fi.Size() >= info.logMaxSize {
					info.fileIndex++
					info.fileLogging.Close()
					info.fileName = logFileName(info.fileIndex, info)
					newLogger()
				} else {
					break
				}
			} else {
				break
			}
		}
	}
	//info.logMu.Unlock()

	switch logLevel {
	case Trace:
		logger.SetPrefix("[TRACE] ")
	case Debug:
		logger.SetPrefix("[DEBUG] ")
	case Info:
		logger.SetPrefix("[INFO] ")
	case Warning:
		logger.SetPrefix("[WARN] ")
	case Error:
		logger.SetPrefix("[ERROR] ")
	case Fatal:
		logger.SetPrefix("[FATAL] ")
	}
	info.logMu.Unlock()

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	//logMu.Lock()

	strbuf := bytes.NewBufferString(path.Base(file))
	strbuf.WriteString(":")
	strbuf.WriteString(strconv.Itoa(line))
	strbuf.WriteString(" ")
	strbuf.WriteString(logFmt)
	logFmt = strbuf.String()

	//logFmt = fmt.Sprintf("%v:%v %v",file, line, logFmt)

	//logMu.Unlock()
	logger.Printf(logFmt, v...)

	//for console
	//log.SetPrefix(logger.Prefix())
	log.Printf(logFmt, v...)
}

func specialWriteLog(info *LogStInfo, logFmt string, v ...interface{}) {
	if info.path == "" {
		return
	}
	info.logMu.Lock()
	newFileName := logFileName(0, info)
	if info.fileName != newFileName {
		if info.fileLogging != nil {
			info.fileLogging.Close()
		}
		info.fileName = newFileName
		info.fileIndex = 0
		newUniLogger()
	} else {
		for {
			fi, err := info.fileLogging.Stat()
			if err == nil {
				if fi.Size() >= info.logMaxSize {
					info.fileIndex++
					info.fileLogging.Close()
					info.fileName = logFileName(info.fileIndex, info)
					newUniLogger()
				} else {
					break
				}
			} else {
				break
			}
		}
	}
	info.logMu.Unlock()

	uniLogger.SetPrefix("")
	uniLogger.Printf(logFmt, v...)

	log.Printf(logFmt, v...)
}
