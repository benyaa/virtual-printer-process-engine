package engine

import (
	"context"
	"github.com/alitto/pond"
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/repo"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"path"
)

type Engine struct {
	Handlers             []handlerContext
	ctx                  context.Context
	filesChannel         chan definitions.PrintInfo
	contentsDir          string
	writeAheadLogger     *repo.WriteAheadLogger
	IgnoreRecoveryErrors bool
	workerPool           *pond.WorkerPool
}

type handlerContext struct {
	handler        definitions.Handler
	retryMechanism config.HandlerRetryMechanism
}

func New(ctx context.Context, config config.Config, files chan definitions.PrintInfo, writeAheadLogger *repo.WriteAheadLogger) *Engine {
	handlers := getHandlers(config)

	return &Engine{
		Handlers:             handlers,
		ctx:                  ctx,
		filesChannel:         files,
		contentsDir:          path.Join(config.Workdir, "contents"),
		writeAheadLogger:     writeAheadLogger,
		IgnoreRecoveryErrors: config.Engine.IgnoreRecoveryErrors,
		workerPool:           pond.New(config.Engine.MaxWorkers, config.Engine.MaxWorkers),
	}
}

func (e *Engine) Recover() error {
	log.Debugf("recovering from WriteAheadLogger")
	entries, err := e.writeAheadLogger.ReadEntries()
	if err != nil {
		return err
	}
	log.Debugf("read %d entries from WriteAheadLogger", len(entries))

	if entries == nil || len(entries) == 0 {
		log.Info("no entries found in WriteAheadLogger; nothing to recover.")
		return nil
	}

	// Map to track sessions and their last log entry
	sessionMap := make(map[uuid.UUID]repo.LogEntry)

	for _, entry := range entries {
		// If the session is marked as ended, remove it from the session map
		if entry.HandlerName == "__end__" {
			delete(sessionMap, entry.SessionID)
			continue
		}
		sessionMap[entry.SessionID] = entry
	}

	for sessionID, lastEntry := range sessionMap {
		log.Debugf("recovering session %s starting from handler %s", sessionID, lastEntry.HandlerID)
		var fileHandler *DefaultEngineFileHandler

		// If the last handler was "__init__", start from the beginning
		if lastEntry.HandlerID == "__init__" {
			log.Debugf("last entry for session %s was '__init__'", sessionID)
			err = utils.CopyFile(lastEntry.InputFile, lastEntry.OutputFile)
			if err != nil {
				log.WithError(err).Errorf("failed to recover during __init__ CopyFile operation from %s to %s", lastEntry.InputFile, lastEntry.OutputFile)
				if !e.IgnoreRecoveryErrors {
					return err
				}
				continue
			}
			fileHandler = NewDefaultEngineFileHandler(lastEntry.OutputFile)
		} else {
			fileHandler = NewDefaultEngineFileHandler(lastEntry.InputFile)
		}

		// Recover the flow object and resume processing
		flow := lastEntry.FlowObject
		err = e.processHandlers(&flow, fileHandler, lastEntry.HandlerID, sessionID)
		if err != nil && !e.IgnoreRecoveryErrors {
			log.WithError(err).Errorf("failed to recover session %s", sessionID)
			return err
		}
	}

	log.Info("Recovery completed.")
	return nil
}

func (e *Engine) Run() {
	err := e.Recover()
	if err != nil && !e.IgnoreRecoveryErrors {
		log.WithError(err).Error("failed to recover, if you don't want to recover, please delete the WAL file or set ignore_recovery_errors to true")
		panic(err)
	}
	for {
		select {
		case <-e.ctx.Done():
			e.workerPool.Stop()
			return
		case i := <-e.filesChannel:
			e.workerPool.Submit(func() {
				e.handleFile(i)
			})
		}
	}
}

func (e *Engine) handleFile(i definitions.PrintInfo) {
	var err error
	sessionID := uuid.New()
	flow := &definitions.EngineFlowObject{
		Pages:    i.Pages,
		Metadata: map[string]interface{}{},
	}
	input := path.Join(e.contentsDir, uuid.NewString())

	walEntry := repo.LogEntry{
		SessionID:   sessionID,
		HandlerName: "__init__",
		HandlerID:   "__init__",
		InputFile:   i.Filepath,
		OutputFile:  input,
		FlowObject:  *flow,
	}
	e.writeAheadLogger.WriteEntry(walEntry)
	err = utils.CopyFile(i.Filepath, input)
	if err != nil {
		log.WithError(err).Errorf("failed to copy file %s to contents folder", i.Filepath)
		return
	}

	fileHandler := NewDefaultEngineFileHandler(input)

	err = e.processHandlers(flow, fileHandler, "", sessionID)
	if err != nil {
		log.WithError(err).Error("failed to process handlers")
		return
	}
}
