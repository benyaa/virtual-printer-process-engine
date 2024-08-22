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
	writeAheadLogger     repo.WriteAheadLogger
	IgnoreRecoveryErrors bool
	workerPool           *pond.WorkerPool
}

type handlerContext struct {
	handler        definitions.Handler
	retryMechanism config.HandlerRetryMechanism
}

func New(ctx context.Context, config config.Config, files chan definitions.PrintInfo, writeAheadLogger repo.WriteAheadLogger) *Engine {
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

func (e *Engine) Run() {
	err := e.Recover()
	if err != nil && !e.IgnoreRecoveryErrors {
		log.WithError(err).Error("failed to recover, if you don't want to recover, please delete the WAL file or set ignore_recovery_errors to true")
		panic(err)
	}
	for {
		select {
		case <-e.ctx.Done():
			log.Infof("stopping engine")
			e.workerPool.Stop()
			return
		case i := <-e.filesChannel:
			log.Debugf("received file %s", i.Filepath)
			e.workerPool.Submit(func() {
				e.handleFile(i)
			})
		}
	}
}

func (e *Engine) handleFile(i definitions.PrintInfo) {
	var err error
	sessionID := uuid.New()
	log.Debugf("handling file %s with sessionID %s", i.Filepath, sessionID)
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
	log.Debugf("writing WAL entry for handler __init__")
	e.writeAheadLogger.WriteEntry(walEntry)
	log.Debugf("copying file %s to contents folder", i.Filepath)
	err = utils.CopyFile(i.Filepath, input)
	if err != nil {
		log.WithError(err).Errorf("failed to copy file %s to contents folder", i.Filepath)
		return
	}
	log.Debugf("copied file %s to contents folder", i.Filepath)

	fileHandler := NewDefaultEngineFileHandler(input)

	log.Debugf("processing handlers")
	err = e.processHandlers(flow, fileHandler, "", sessionID)
	if err != nil {
		log.WithError(err).Error("failed to process handlers")
		return
	}
}
