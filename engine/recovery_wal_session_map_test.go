package engine

import (
	"testing"

	"github.com/benyaa/virtual-printer-process-engine/repo"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateSessionMapForWAL_NoEntries(t *testing.T) {
	engine := Engine{}
	entries := []repo.LogEntry{}

	sessionMap := engine.createSessionMapForWAL(entries)

	assert.Empty(t, sessionMap)
}

func TestCreateSessionMapForWAL_SingleCompletedSession(t *testing.T) {
	engine := Engine{}
	sessionID := uuid.New()
	entries := []repo.LogEntry{
		{SessionID: sessionID, HandlerName: "__init__"},
		{SessionID: sessionID, HandlerName: "__end__"},
	}

	sessionMap := engine.createSessionMapForWAL(entries)

	assert.Empty(t, sessionMap)
}

func TestCreateSessionMapForWAL_SingleIncompleteSession(t *testing.T) {
	engine := Engine{}
	sessionID := uuid.New()
	entries := []repo.LogEntry{
		{SessionID: sessionID, HandlerName: "__init__"},
		{SessionID: sessionID, HandlerName: "handler_1"},
	}

	sessionMap := engine.createSessionMapForWAL(entries)

	assert.Len(t, sessionMap, 1)
	assert.Contains(t, sessionMap, sessionID)
	assert.Equal(t, "handler_1", sessionMap[sessionID].HandlerName)
}

func TestCreateSessionMapForWAL_MultipleSessionsMixed(t *testing.T) {
	engine := Engine{}
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()
	entries := []repo.LogEntry{
		{SessionID: sessionID1, HandlerName: "__init__"},
		{SessionID: sessionID1, HandlerName: "__end__"},
		{SessionID: sessionID2, HandlerName: "__init__"},
		{SessionID: sessionID2, HandlerName: "handler_1"},
	}

	sessionMap := engine.createSessionMapForWAL(entries)

	assert.Len(t, sessionMap, 1)
	assert.Contains(t, sessionMap, sessionID2)
	assert.Equal(t, "handler_1", sessionMap[sessionID2].HandlerName)
}
