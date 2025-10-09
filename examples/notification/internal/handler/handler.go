package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"notification-service/internal/query"
	"notification-service/internal/sse"

	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

const APIPrefix = "/api"

// API wraps the generated queries with HTTP handlers.
type API struct {
	DB         *sql.DB
	SSEManager *sse.Manager
}

// New returns a new API instance.
func New(db *sql.DB) *API {
	return &API{
		DB:         db,
		SSEManager: sse.NewManager(),
	}
}

// Register binds handlers to the provided mux.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+APIPrefix+"/users/{userID}/notifications", a.handleListUserNotifications)
	mux.HandleFunc("GET "+APIPrefix+"/notifications/users/{userID}/stream", a.handleSSEStream)
	mux.HandleFunc("POST "+APIPrefix+"/notifications", a.handleCreateNotification)
	mux.HandleFunc("PATCH "+APIPrefix+"/notifications/{id}", a.handleUpdateNotification)
	mux.HandleFunc("POST "+APIPrefix+"/notifications/{id}/cancel", a.handleCancelNotification)
	mux.HandleFunc("POST "+APIPrefix+"/users/{userID}/notifications/{notificationID}/read", a.handleMarkAsRead)
	mux.HandleFunc("POST "+APIPrefix+"/users/{userID}/notifications/mark-non-important", a.handleMarkNonImportantAsRead)
	mux.HandleFunc("POST "+APIPrefix+"/maintenance/notifications/delete-old", a.handleDeleteOldNotifications)
}

// -----------------------------------------------------------------------------
// Request / response payloads

type listNotificationsRequest struct {
	UnreadOnly bool      `json:"unread_only,omitempty"`
	Since      time.Time `json:"since,omitempty"`
}

type notificationItem struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	IconURL     *string    `json:"icon_url,omitempty"`
	Important   bool       `json:"important"`
	Cancelable  bool       `json:"cancelable"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

type createNotificationRequest struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Important  bool     `json:"important"`
	Cancelable bool     `json:"cancelable"`
	IconURL    *string  `json:"icon_url"`
	ExpiresAt  *string  `json:"expires_at"`
	UserIDs    []string `json:"user_ids"`
}

type createNotificationResponse struct {
	ID               int        `json:"id"`
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	Important        bool       `json:"important"`
	Cancelable       bool       `json:"cancelable"`
	IconURL          *string    `json:"icon_url,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	CreatedAt        *time.Time `json:"created_at"`
	DeliveredUserIDs []string   `json:"delivered_user_ids,omitempty"`
	DeliveredCount   int        `json:"delivered_count,omitempty"`
}

type updateNotificationRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Important bool   `json:"important"`
}

type updateNotificationResponse struct {
	ID         int        `json:"id"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Important  bool       `json:"important"`
	Cancelable bool       `json:"cancelable"`
	IconURL    *string    `json:"icon_url,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  *time.Time `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

type cancelNotificationRequest struct {
	CancelMessage string `json:"cancel_message"`
}

type deleteOldNotificationsRequest struct {
	Before string `json:"before"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// -----------------------------------------------------------------------------
// Handlers

func (a *API) handleListUserNotifications(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if strings.TrimSpace(userID) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "userID is required"})
		return
	}

	req, limit := parseListRequest(r)
	ctx := snapsqlgo.WithSystemValue(r.Context(), "limit", limit)

	if req.Since.IsZero() {
		log.Printf("[LIST] User: %s, UnreadOnly: %v, Since: (none - initial request)", userID, req.UnreadOnly)
	} else {
		log.Printf("[LIST] User: %s, UnreadOnly: %v, Since: %v", userID, req.UnreadOnly, req.Since.Format(time.RFC3339))
	}

	seq := query.ListUserNotifications(ctx, a.DB, userID, req.UnreadOnly, req.Since)
	items := make([]notificationItem, 0)

	for res, err := range seq {
		if err != nil {
			log.Printf("list notifications failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list notifications"})

			return
		}

		if res == nil {
			continue
		}

		items = append(items, mapListResult(res))
	}

	log.Printf("[LIST] Returning %d notification(s) for user %s", len(items), userID)
	if len(items) > 0 {
		for _, item := range items {
			var updatedAt string
			if item.UpdatedAt != nil {
				updatedAt = item.UpdatedAt.Format(time.RFC3339)
			} else {
				updatedAt = "(nil)"
			}
			log.Printf("[LIST]   - ID: %d, Title: %s, CreatedAt: %v, UpdatedAt: %s",
				item.ID, item.Title, item.CreatedAt.Format(time.RFC3339), updatedAt)
		}
	}

	writeJSON(w, http.StatusOK, items)
}

func (a *API) handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())

	var req createNotificationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "title is required"})
		return
	}

	if strings.TrimSpace(req.Body) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "body is required"})
		return
	}

	if len(req.UserIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_ids must not be empty"})
		return
	}

	expireStr := valueOrEmpty(req.ExpiresAt)

	var expires time.Time

	if expireStr != "" {
		var err error

		parsedExpires, err := parseTimestamp(expireStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid expires_at timestamp"})
			return
		}

		if expires.Before(time.Now()) {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "expires_at must be in the future"})
			return
		}

		expires = parsedExpires
	}

	icon := valueOrEmpty(req.IconURL)

	log.Println("🦆", req.Title, req.Body, req.Important, req.Cancelable, icon, expires)

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("create notification begin tx failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create notification"})

		return
	}

	rollback := true

	defer func() {
		if rollback {
			if err := tx.Rollback(); err != nil {
				log.Printf("create notification rollback failed: %v", err)
			}
		}
	}()

	res, err := query.CreateNotification(ctx, tx, req.Title, req.Body, req.Important, req.Cancelable, icon, expires)
	log.Println(res, err)

	if err != nil {
		log.Printf("create notification failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create notification"})

		return
	}

	log.Println(res.ID, res.Title, res.Body)

	deliveredCount := 0

	if len(req.UserIDs) > 0 {
		if _, err := query.DeliverNotification(ctx, tx, res.ID, req.UserIDs, nil, nil); err != nil {
			log.Printf("deliver notification during create failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to deliver notification"})

			return
		}

		deliveredCount = len(req.UserIDs)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("create notification commit failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create notification"})

		return
	}

	rollback = false

	reply := createNotificationResponse{
		ID:               res.ID,
		Title:            res.Title,
		Body:             res.Body,
		Important:        derefBool(res.Important),
		Cancelable:       derefBool(res.Cancelable),
		IconURL:          res.IconUrl,
		ExpiresAt:        res.ExpiresAt,
		CreatedAt:        res.CreatedAt,
		DeliveredUserIDs: append([]string(nil), req.UserIDs...),
		DeliveredCount:   deliveredCount,
	}

	// Broadcast notification event to SSE clients
	notification := notificationItem{
		ID:          res.ID,
		Title:       res.Title,
		Body:        res.Body,
		IconURL:     res.IconUrl,
		Important:   derefBool(res.Important),
		Cancelable:  derefBool(res.Cancelable),
		ExpiresAt:   res.ExpiresAt,
		CreatedAt:   res.CreatedAt,
		DeliveredAt: res.CreatedAt,
	}

	event := sse.Event{
		Type:    "notification",
		Payload: notification,
	}

	a.SSEManager.BroadcastToMultiple(req.UserIDs, event)

	writeJSON(w, http.StatusCreated, reply)
}

func (a *API) handleUpdateNotification(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())

	notificationID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid notification id"})
		return
	}

	var req updateNotificationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "title is required"})
		return
	}

	if strings.TrimSpace(req.Body) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "body is required"})
		return
	}

	// Start transaction to update notification and clear read_at
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("update notification begin tx failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update notification"})
		return
	}

	rollback := true
	defer func() {
		if rollback {
			if err := tx.Rollback(); err != nil {
				log.Printf("update notification rollback failed: %v", err)
			}
		}
	}()

	seq := query.UpdateNotification(ctx, tx, notificationID, req.Title, req.Body, req.Important)
	var result *query.UpdateNotificationResult

	for res, err := range seq {
		if err != nil {
			log.Printf("update notification failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update notification"})
			return
		}

		result = res
		break
	}

	if result == nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "notification not found"})
		return
	}

	// Clear read_at for all users who have this notification
	log.Printf("Clearing read_at for notification ID: %d", notificationID)
	if _, err := query.UnreadNotificationForUsers(ctx, tx, notificationID); err != nil {
		log.Printf("unread notification for users failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update notification"})
		return
	}
	log.Printf("Successfully cleared read_at for notification ID: %d", notificationID)

	if err := tx.Commit(); err != nil {
		log.Printf("update notification commit failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update notification"})
		return
	}

	rollback = false
	log.Printf("Successfully updated notification ID: %d (title: %s, important: %v)", notificationID, req.Title, req.Important)

	reply := updateNotificationResponse{
		ID:         result.ID,
		Title:      result.Title,
		Body:       result.Body,
		Important:  derefBool(result.Important),
		Cancelable: derefBool(result.Cancelable),
		IconURL:    result.IconUrl,
		ExpiresAt:  result.ExpiresAt,
		CreatedAt:  result.CreatedAt,
		UpdatedAt:  result.UpdatedAt,
	}

	// Broadcast update event to SSE clients
	// We need to get the list of users who have this notification
	// For now, we'll broadcast to all connected clients
	// In a production system, you'd query the inbox table to get affected users
	notification := notificationItem{
		ID:         result.ID,
		Title:      result.Title,
		Body:       result.Body,
		IconURL:    result.IconUrl,
		Important:  derefBool(result.Important),
		Cancelable: derefBool(result.Cancelable),
		ExpiresAt:  result.ExpiresAt,
		CreatedAt:  result.CreatedAt,
	}

	event := sse.Event{
		Type:    "update",
		Payload: notification,
	}

	// Get affected users from inbox table
	affectedUsers := a.getAffectedUsers(ctx, notificationID)
	a.SSEManager.BroadcastToMultiple(affectedUsers, event)

	writeJSON(w, http.StatusOK, reply)
}

func (a *API) handleCancelNotification(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())

	notificationID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid notification id"})
		return
	}

	var req cancelNotificationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if strings.TrimSpace(req.CancelMessage) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cancel_message is required"})
		return
	}

	if _, err := query.CancelNotification(ctx, a.DB, notificationID, req.CancelMessage); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to cancel notification"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())
	userID := r.PathValue("userID")

	notificationID, err := strconv.Atoi(r.PathValue("notificationID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid notification id"})
		return
	}

	if strings.TrimSpace(userID) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "userID is required"})
		return
	}

	if _, err := query.MarkAsRead(ctx, a.DB, notificationID, userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to mark notification as read"})
		return
	}

	// Broadcast read event to SSE client
	readEvent := map[string]interface{}{
		"notification_id": notificationID,
		"user_id":         userID,
	}

	event := sse.Event{
		Type:    "read",
		Payload: readEvent,
	}

	a.SSEManager.Broadcast(userID, event)

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMarkNonImportantAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())

	userID := r.PathValue("userID")
	if strings.TrimSpace(userID) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "userID is required"})
		return
	}

	if _, err := query.MarkNonImportantAsRead(ctx, a.DB, userID); err != nil {
		log.Printf("mark non-important as read failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to mark notifications"})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleDeleteOldNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())

	var req deleteOldNotificationsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if strings.TrimSpace(req.Before) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "before is required"})
		return
	}

	before, err := parseTimestamp(req.Before)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid before timestamp"})
		return
	}

	if _, err := query.DeleteOldAndExpiredNotifications(ctx, a.DB, before); err != nil {
		log.Printf("delete old notifications failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete notifications"})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if strings.TrimSpace(userID) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "userID is required"})
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Create SSE client
	client := &sse.Client{
		UserID: userID,
		Chan:   make(chan []byte, 10),
		Done:   make(chan struct{}),
	}

	// Register client
	a.SSEManager.AddClient(userID, client)
	defer a.SSEManager.RemoveClient(userID, client)

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("SSE: streaming not supported")
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Printf("SSE: stream started for user %s", userID)

	for {
		select {
		case <-r.Context().Done():
			log.Printf("SSE: client disconnected (context done) for user %s", userID)
			return

		case <-client.Done:
			log.Printf("SSE: client done for user %s", userID)
			return

		case message := <-client.Chan:
			if _, err := w.Write(message); err != nil {
				log.Printf("SSE: write failed for user %s: %v", userID, err)
				return
			}
			flusher.Flush()

		case <-ticker.C:
			// Send keep-alive comment
			if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
				log.Printf("SSE: keep-alive write failed for user %s: %v", userID, err)
				return
			}
			flusher.Flush()
		}
	}
}

// -----------------------------------------------------------------------------
// Helpers

func parseListRequest(r *http.Request) (listNotificationsRequest, int) {
	values := r.URL.Query()

	unreadOnly := false

	if qs := values.Get("unread_only"); qs != "" {
		if parsed, err := strconv.ParseBool(qs); err == nil {
			unreadOnly = parsed
		}
	}

	since := values.Get("since")

	parsedSince, err := time.Parse(time.RFC3339, since)
	if err != nil {
		parsedSince = time.Time{}
	}

	limit := 1000 // default limit
	if qs := values.Get("limit"); qs != "" {
		if parsed, err := strconv.Atoi(qs); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	return listNotificationsRequest{
		UnreadOnly: unreadOnly,
		Since:      parsedSince,
	}, limit
}

func mapListResult(res *query.ListUserNotificationsResult) notificationItem {
	return notificationItem{
		ID:          res.NID,
		Title:       res.NTitle,
		Body:        res.NBody,
		IconURL:     res.NIconUrl,
		Important:   derefBool(res.NImportant),
		Cancelable:  derefBool(res.NCancelable),
		ExpiresAt:   res.NExpiresAt,
		CreatedAt:   res.NCreatedAt,
		UpdatedAt:   res.NUpdatedAt,
		ReadAt:      res.IReadAt,
		DeliveredAt: res.DeliveredAt,
	}
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("request body is empty")
		}

		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			log.Printf("write response failed: %v", err)
		}
	}
}

func derefBool(v *bool) bool {
	if v == nil {
		return false
	}

	return *v
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

func parseTimestamp(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported timestamp format")
}

func contextWithSystemDefaults(parent context.Context) context.Context {
	now := time.Now().UTC()
	ctx := snapsqlgo.WithSystemValue(parent, "created_at", now)
	ctx = snapsqlgo.WithSystemValue(ctx, "updated_at", now)

	return ctx
}

func (a *API) getAffectedUsers(ctx context.Context, notificationID int) []string {
	// Query the inbox table to get all users who have this notification
	query := `SELECT user_id FROM inbox WHERE notification_id = $1`
	rows, err := a.DB.QueryContext(ctx, query, notificationID)
	if err != nil {
		log.Printf("failed to get affected users: %v", err)
		return []string{}
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			log.Printf("failed to scan user id: %v", err)
			continue
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs
}
