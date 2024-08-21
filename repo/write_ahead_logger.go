package repo

import (
	"bufio"
	"encoding/json"
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

type LogEntry struct {
	SessionID   uuid.UUID                    `json:"session_id"`
	HandlerName string                       `json:"handler_name"`
	HandlerID   string                       `json:"handler_id"`
	InputFile   string                       `json:"input_file"`
	OutputFile  string                       `json:"output_file"`
	FlowObject  definitions.EngineFlowObject `json:"flow_object"`
}

type WriteAheadLogger interface {
	WriteEntry(entry LogEntry)
	ReadEntries() ([]LogEntry, error)
}

type DefaultWriteAheadLogger struct {
	logger   *log.Logger
	filePath string
	enabled  bool
}

func NewWriteAheadLogger(logFilePath string, conf config.WriteAheadLogging) WriteAheadLogger {
	walLogger := log.New()

	walLogger.Out = &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    conf.MaxSizeMB,
		MaxBackups: conf.MaxBackups,
		MaxAge:     conf.MaxAgeDays,
		Compress:   true,
	}

	walLogger.SetFormatter(&log.JSONFormatter{})
	walLogger.SetLevel(log.InfoLevel)

	return &DefaultWriteAheadLogger{
		logger:   walLogger,
		filePath: logFilePath,
		enabled:  conf.Enabled,
	}
}

func (l *DefaultWriteAheadLogger) ReadEntries() ([]LogEntry, error) {
	if !l.enabled {
		return nil, nil
	}
	log.Debugf("reading WAL entries from %s", l.filePath)

	if _, err := os.Stat(l.filePath); os.IsNotExist(err) {
		log.Debugf("WAL file %s does not exist", l.filePath)
		return nil, nil
	}

	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, err
	}
	log.Debugf("opened WAL file %s", l.filePath)
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry LogEntry
		err := json.Unmarshal(scanner.Bytes(), &entry)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	log.Debugf("read %d WAL entries", len(entries))

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	log.Debugf("finished reading WAL entries")

	return entries, nil

}

func (l *DefaultWriteAheadLogger) WriteEntry(entry LogEntry) {
	if !l.enabled {
		return
	}
	l.logger.WithFields(log.Fields{
		"session_id":   entry.SessionID.String(),
		"handler_name": entry.HandlerName,
		"handler_id":   entry.HandlerID,
		"input_file":   entry.InputFile,
		"output_file":  entry.OutputFile,
		"flow_object":  entry.FlowObject,
	}).Info("WAL entry recorded")
}
