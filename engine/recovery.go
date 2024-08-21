package engine

import (
	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/repo"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

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
	sessionMap := e.createSessionMapForWAL(entries)

	for sessionID, lastEntry := range sessionMap {
		fileHandler, flow, err := e.getProcessHandlerForSession(sessionID, lastEntry, err)
		if err != nil {
			if !e.IgnoreRecoveryErrors {
				return err
			}
			continue
		}

		err = e.processHandlers(flow, fileHandler, lastEntry.HandlerID, sessionID)
		if err != nil && !e.IgnoreRecoveryErrors {
			log.WithError(err).Errorf("failed to recover session %s", sessionID)
			return err
		}
	}
	log.Info("recovery completed.")
	return nil
}

func (e *Engine) createSessionMapForWAL(entries []repo.LogEntry) map[uuid.UUID]repo.LogEntry {
	sessionMap := make(map[uuid.UUID]repo.LogEntry)

	for _, entry := range entries {
		// If the session is marked as ended, remove it from the session map
		if entry.HandlerName == "__end__" {
			delete(sessionMap, entry.SessionID)
			continue
		}
		sessionMap[entry.SessionID] = entry
	}
	return sessionMap
}

func (e *Engine) getProcessHandlerForSession(sessionID uuid.UUID, lastEntry repo.LogEntry, err error) (*DefaultEngineFileHandler, *definitions.EngineFlowObject, error) {
	log.Debugf("recovering session %s starting from handler %s", sessionID, lastEntry.HandlerID)
	var fileHandler *DefaultEngineFileHandler

	// If the last handler was "__init__", start from the beginning
	if lastEntry.HandlerID == "__init__" {
		log.Debugf("last entry for session %s was '__init__'", sessionID)
		err = utils.CopyFile(lastEntry.InputFile, lastEntry.OutputFile)
		if err != nil {
			log.WithError(err).Errorf("failed to recover during __init__ CopyFile operation from %s to %s", lastEntry.InputFile, lastEntry.OutputFile)
			return nil, nil, err
		}
		fileHandler = NewDefaultEngineFileHandler(lastEntry.OutputFile)
	} else {
		fileHandler = NewDefaultEngineFileHandler(lastEntry.InputFile)
	}

	// Recover the flow object and resume processing
	flow := lastEntry.FlowObject
	return fileHandler, &flow, nil
}
