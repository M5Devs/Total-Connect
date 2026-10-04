package core

import (
	"context"
	"sync"

	"github.com/M5Devs/Total-Connect/internal/models"
)

// SessionPane represents the state of a single view/pane in the interface.
type SessionPane struct {
	CurrentPath string
	Items       []models.FileItem
}

// SessionManager manages active panes and interaction with the storage engine.
type SessionManager struct {
	mu     sync.RWMutex
	engine StorageEngine
	panes  map[string]*SessionPane
}

// NewSessionManager creates a new SessionManager with the given StorageEngine.
func NewSessionManager(engine StorageEngine) *SessionManager {
	return &SessionManager{
		engine: engine,
		panes:  make(map[string]*SessionPane),
	}
}

// SetEngine updates or sets the StorageEngine.
func (sm *SessionManager) SetEngine(engine StorageEngine) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.engine = engine
}

// GetEngine returns the current StorageEngine.
func (sm *SessionManager) GetEngine() StorageEngine {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.engine
}

// SetPanePath sets the path for a specific pane ID.
func (sm *SessionManager) SetPanePath(paneID, path string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	p, ok := sm.panes[paneID]
	if !ok {
		p = &SessionPane{}
		sm.panes[paneID] = p
	}
	p.CurrentPath = path
}

// GetPane returns the pane state for a given pane ID.
func (sm *SessionManager) GetPane(paneID string) (*SessionPane, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	p, ok := sm.panes[paneID]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// RefreshPane reloads entries for a pane using the storage engine.
func (sm *SessionManager) RefreshPane(ctx context.Context, paneID string) ([]models.FileItem, error) {
	sm.mu.Lock()
	p, ok := sm.panes[paneID]
	if !ok {
		p = &SessionPane{}
		sm.panes[paneID] = p
	}
	path := p.CurrentPath
	sm.mu.Unlock()

	items, err := sm.engine.ListEntries(ctx, path)
	if err != nil {
		return nil, err
	}

	sm.mu.Lock()
	p.Items = items
	sm.mu.Unlock()

	return items, nil
}
