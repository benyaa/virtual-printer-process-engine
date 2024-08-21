package engine

import (
	"errors"
	"testing"

	"github.com/benyaa/virtual-printer-process-engine/definitions"
	"github.com/benyaa/virtual-printer-process-engine/repo"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetProcessHandlerForSession_InitHandler(t *testing.T) {
	// Mock CopyFile for testing
	originalCopyFile := utils.CopyFile
	utils.CopyFile = func(src, dst string) error {
		return nil
	}
	defer func() { utils.CopyFile = originalCopyFile }() // Restore after test

	sessionID := uuid.New()

	engine := Engine{}
	lastEntry := repo.LogEntry{
		SessionID:   sessionID,
		HandlerName: "__init__",
		HandlerID:   "__init__",
		InputFile:   "input.txt",
		OutputFile:  "output.txt",
		FlowObject:  definitions.EngineFlowObject{},
	}

	fileHandler, flow, err := engine.getProcessHandlerForSession(sessionID, lastEntry, nil)

	assert.NoError(t, err)
	assert.NotNil(t, fileHandler)
	assert.Equal(t, "output.txt", fileHandler.input)
	assert.Equal(t, lastEntry.FlowObject, *flow)
}

func TestGetProcessHandlerForSession_CopyFileError(t *testing.T) {
	// Mock CopyFile to return an error
	originalCopyFile := utils.CopyFile
	utils.CopyFile = func(src, dst string) error {
		return errors.New("copy error")
	}
	defer func() { utils.CopyFile = originalCopyFile }() // Restore after test

	sessionID := uuid.New()

	engine := Engine{}
	lastEntry := repo.LogEntry{
		SessionID:   sessionID,
		HandlerName: "__init__",
		HandlerID:   "__init__",
		InputFile:   "input.txt",
		OutputFile:  "output.txt",
		FlowObject:  definitions.EngineFlowObject{},
	}

	fileHandler, flow, err := engine.getProcessHandlerForSession(sessionID, lastEntry, nil)

	assert.Error(t, err)
	assert.Nil(t, fileHandler)
	assert.Nil(t, flow)
}
