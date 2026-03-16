package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kubestellar/console/pkg/models"
	"github.com/kubestellar/console/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type eventsTestStore struct {
	test.MockStore
	recordedEvent *models.UserEvent
	recordErr     error
}

func (s *eventsTestStore) RecordEvent(event *models.UserEvent) error {
	s.recordedEvent = event
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	return s.recordErr
}

func TestEventRecordEvent_Success(t *testing.T) {
	env := setupTestEnv(t)
	store := &eventsTestStore{}
	handler := NewEventHandler(store)
	env.App.Post("/api/events", handler.RecordEvent)

	cardID := uuid.New()
	body, err := json.Marshal(map[string]any{
		"event_type": string(models.EventTypeCardAction),
		"card_id":    cardID.String(),
		"metadata": map[string]any{
			"source": "dashboard",
			"count":  3,
		},
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req, 5000)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.NotNil(t, store.recordedEvent)
	assert.Equal(t, testAdminUserID, store.recordedEvent.UserID)
	assert.Equal(t, models.EventTypeCardAction, store.recordedEvent.EventType)
	require.NotNil(t, store.recordedEvent.CardID)
	assert.Equal(t, cardID, *store.recordedEvent.CardID)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(store.recordedEvent.Metadata, &metadata))
	assert.Equal(t, "dashboard", metadata["source"])
	assert.Equal(t, float64(3), metadata["count"])
}

func TestEventRecordEvent_InvalidBody(t *testing.T) {
	env := setupTestEnv(t)
	handler := NewEventHandler(&eventsTestStore{})
	env.App.Post("/api/events", handler.RecordEvent)

	req, err := http.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString("{"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req, 5000)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestEventRecordEvent_StoreError(t *testing.T) {
	env := setupTestEnv(t)
	store := &eventsTestStore{recordErr: errors.New("write failed")}
	handler := NewEventHandler(store)
	env.App.Post("/api/events", handler.RecordEvent)

	body, err := json.Marshal(map[string]any{"event_type": string(models.EventTypePageView)})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.App.Test(req, 5000)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
