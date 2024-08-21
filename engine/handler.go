package engine

import (
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/handler"
	"github.com/benyaa/virtual-printer-process-engine/repo"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func getHandlers(config config.Config) []handlerContext {
	var handlers []handlerContext
	previousID := ""
	for _, currentHandler := range config.Engine.Handlers {
		h, err := handler.GetHandler(currentHandler, previousID)
		if err != nil {
			log.WithError(err).Errorf("failed to get handler %s", currentHandler.Name)
			panic(err)
		}
		previousID = h.GetID()
		retry := currentHandler.Retry
		initRetryDefaults(&retry)
		handlers = append(handlers, handlerContext{
			handler:        h,
			retryMechanism: retry,
		})
	}
	return handlers
}

func initRetryDefaults(retry *config.HandlerRetryMechanism) {
	if retry.MaxRetries == 0 {
		retry.MaxRetries = 1
	}
}

func (e *Engine) processHandlers(flow *definitions.EngineFlowObject, fileHandler *DefaultEngineFileHandler, startHandlerID string, sessionID uuid.UUID) error {
	log.Tracef("processing handlers")
	resume := startHandlerID == ""
	log.Debugf("resuming from handler %s", startHandlerID)

	for _, hCtx := range e.Handlers {
		h := hCtx.handler
		handlerID := h.GetID()
		if handlerID == startHandlerID && !resume {
			resume = true
		}
		if resume {
			log.Debugf("handling %s with handler %s", fileHandler.input, h.Name())
			logEntry := repo.LogEntry{
				SessionID:   sessionID,
				HandlerName: h.Name(),
				HandlerID:   handlerID,
				InputFile:   fileHandler.input,
				OutputFile:  fileHandler.output,
				FlowObject:  *flow,
			}
			log.Debugf("writing WAL entry for handler %s (%s)", h.Name(), handlerID)
			e.writeAheadLogger.WriteEntry(logEntry)
			log.Debugf("deep copying flow object for handler %s (%s)", h.Name(), handlerID)

			copiedFlow, err := utils.DeepCopy(flow)
			if err != nil {
				log.WithError(err).Error("failed to copy flow object")
				return err
			}

			log.Debugf("handling %s with handler %s", fileHandler.input, h.Name())

			retryMechanism := hCtx.retryMechanism
			for attempts := 1; attempts <= retryMechanism.MaxRetries; attempts++ {
				log.Debugf("attempt %d/%d", attempts, retryMechanism.MaxRetries)
				newFlow, err := h.Handle(copiedFlow, fileHandler)
				if err != nil {
					if attempts < retryMechanism.MaxRetries {
						log.WithError(err).Warnf("retrying handler %s (%d/%d)", h.Name(), attempts+1, retryMechanism.MaxRetries)
						time.Sleep(time.Duration(retryMechanism.BackOffInterval) * time.Second)
					} else {
						log.WithError(err).Errorf("failed to handle %s with handler %s after %d attempts", fileHandler.input, h.Name(), retryMechanism.MaxRetries)
						return err
					}
				} else {
					flow = newFlow
					break
				}
			}
			log.Debugf("handled %s with handler %s", fileHandler.input, h.Name())

			fileHandler = fileHandler.getNewFileHandler()
		}

	}

	if !resume {
		log.Warnf("no handlers were processed, the engine will not write the output file")
	}
	logEntry := repo.LogEntry{
		SessionID:   sessionID,
		HandlerName: "__end__",
		HandlerID:   "__end__",
		InputFile:   fileHandler.input,
		OutputFile:  fileHandler.output,
		FlowObject:  *flow,
	}
	e.writeAheadLogger.WriteEntry(logEntry)
	err := os.Remove(fileHandler.input)
	if err != nil {
		log.WithError(err).Warnf("failed to remove final input file %s", fileHandler.input)
	}

	log.Info("finished processing handlers for file %s", fileHandler.input)

	return nil
}
