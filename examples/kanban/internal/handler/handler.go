package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shibukawa/snapsql/examples/kanban/internal/query"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// Common errors
var (
	ErrValidation = errors.New("validation error")
)

// withTransaction executes a function within a database transaction
func withTransaction(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	err = fn(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Request structs
type (
	BoardCreateRequest struct {
		Name string `json:"name"`
	}

	CardCreateRequest struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Position    float64 `json:"position"`
	}

	CardUpdateRequest struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	CardMoveRequest struct {
		TargetListID   int     `json:"target_list_id"`
		TargetPosition float64 `json:"target_position"`
	}

	CardReorderRequest struct {
		Position float64 `json:"position"`
	}

	ListArchiveRequest struct {
		IsArchived bool `json:"is_archived"`
	}

	ListRenameRequest struct {
		Name string `json:"name"`
	}

	ListReorderRequest struct {
		Position float64 `json:"position"`
	}

	CardCommentCreateRequest struct {
		Text string `json:"text"`
	}
)

// API provides HTTP handlers for kanban operations.
// This is a thin layer that:
// 1. Parses inputs from URL/path/body
// 2. Calls generated query functions
// 3. Translates results into JSON (currently placeholder because generated functions return sql.Result)
// Later we can adjust once code generator emits typed result structs.
type API struct {
	DB *sql.DB
}

const APIPrefix = "/api"

const (
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

// New creates a new API instance.
func New(db *sql.DB) *API { return &API{DB: db} }

// Register attaches handlers to the given mux.
func (a *API) Register(mux *http.ServeMux) {
	// Board endpoints
	mux.HandleFunc("GET "+APIPrefix+"/boards", a.handleBoardList)
	mux.HandleFunc("POST "+APIPrefix+"/boards", a.handleBoardCreate)
	mux.HandleFunc("GET "+APIPrefix+"/boards/{id}", a.handleBoardGet)
	mux.HandleFunc("GET "+APIPrefix+"/boards/{id}/tree", a.handleBoardTree)

	// List endpoints
	mux.HandleFunc("POST "+APIPrefix+"/lists", a.handleListCreate)
	mux.HandleFunc("POST "+APIPrefix+"/lists/{id}/archive", a.handleListArchive)
	mux.HandleFunc("POST "+APIPrefix+"/lists/{id}/rename", a.handleListRename)
	mux.HandleFunc("POST "+APIPrefix+"/lists/{id}/reorder", a.handleListReorder)

	// Card endpoints
	mux.HandleFunc("POST "+APIPrefix+"/cards", a.handleCardCreate)
	mux.HandleFunc("PATCH "+APIPrefix+"/cards/{id}", a.handleCardUpdate)
	mux.HandleFunc("POST "+APIPrefix+"/cards/{id}/move", a.handleCardMove)
	mux.HandleFunc("POST "+APIPrefix+"/cards/{id}/reorder", a.handleCardReorder)
	mux.HandleFunc("GET "+APIPrefix+"/cards/{id}/comments", a.handleCardCommentList)
	mux.HandleFunc("POST "+APIPrefix+"/cards/{id}/comments", a.handleCardCommentCreate)
}

// writeJSON writes data as JSON with proper headers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	if status >= http.StatusBadRequest {
		logErrorStatus(status, v)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// errorResponse represents a simple error payload.
type errorResponse struct {
	Error string `json:"error"`
}

func logErrorStatus(status int, payload any) {
	message := extractErrorMessage(payload)
	if message == "" {
		if raw, err := json.Marshal(payload); err == nil {
			message = string(raw)
		} else {
			message = fmt.Sprintf("%v", payload)
		}
	}

	log.Printf("%sHTTP %d: %s%s", colorRed, status, message, colorReset)
}

func extractErrorMessage(payload any) string {
	switch value := payload.(type) {
	case errorResponse:
		return value.Error
	case *errorResponse:
		if value != nil {
			return value.Error
		}
	case map[string]any:
		if msg, ok := value["error"].(string); ok {
			return msg
		}
	case *map[string]any:
		if value != nil {
			if msg, ok := (*value)["error"].(string); ok {
				return msg
			}
		}
	}

	return ""
}

func (a *API) handleBoardList(w http.ResponseWriter, r *http.Request) {
	ctx := contextWithSystemDefaults(r.Context())
	seq := query.BoardList(ctx, a.DB)
	results := make([]query.BoardListResult, 0)

	for item, err := range seq {
		if err != nil {
			log.Printf("board list failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		if item != nil {
			results = append(results, *item)
		}
	}

	writeJSON(w, http.StatusOK, results)
}

func (a *API) handleBoardCreate(w http.ResponseWriter, r *http.Request) {
	var req BoardCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: name is required", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	var board query.BoardCreateResult

	var lists []query.ListCreateResult

	err := withTransaction(ctx, a.DB, func(tx *sql.Tx) error {
		// Archive existing active board and get the archived board ID
		var archivedBoardID int

		seq := query.BoardArchive(ctx, tx)
		for archivedBoard, err := range seq {
			if err != nil {
				return fmt.Errorf("board archive failed: %w", err)
			}

			if archivedBoard != nil {
				archivedBoardID = archivedBoard.ID
			}
		}

		// Create new board
		var err error

		board, err = query.BoardCreate(ctx, tx, req.Name)
		if err != nil {
			return fmt.Errorf("board create failed: %w", err)
		}

		// Create lists from templates
		lists, err = createListsFromTemplatesWithTx(ctx, tx, board.ID)
		if err != nil {
			return fmt.Errorf("list creation failed: %w", err)
		}

		// Move incomplete cards from archived board if any exist
		if archivedBoardID != 0 {
			if _, err := query.CardPostpone(ctx, tx, archivedBoardID, board.ID); err != nil {
				log.Printf("card postpone failed: %v", err)
				// Continue anyway - this is not critical
			}
		}

		return nil
	})
	if err != nil {
		log.Printf("board create transaction failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

		return
	}

	// Return board with created lists
	response := map[string]any{
		"board": board,
		"lists": lists,
	}
	writeJSON(w, http.StatusCreated, response)
}

// handleBoardGet handles GET /boards/{id}
func (a *API) handleBoardGet(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid board id", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	board, err := query.BoardGet(ctx, a.DB, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "board not found"})
			return
		}

		log.Printf("board get failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

		return
	}

	writeJSON(w, http.StatusOK, board)
}

// handleBoardTree handles GET /boards/{id}/tree
func (a *API) handleBoardTree(w http.ResponseWriter, r *http.Request) {
	boardID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid board id", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	result, err := query.BoardTree(ctx, a.DB, boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "board not found"})
			return
		}

		log.Printf("board tree failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleListCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "")
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "list creation is disabled"})
}

func (a *API) handleListArchive(w http.ResponseWriter, r *http.Request) {
	listID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid list id", ErrValidation).Error()})
		return
	}

	var req ListArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	seq := query.ListArchive(ctx, a.DB, listID, req.IsArchived)
	archived := false

	for _, err := range seq {
		if err != nil {
			log.Printf("list archive failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		archived = true
	}

	if !archived {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "list not found"})
		return
	}

	writeNoContent(w)
}

func (a *API) handleListRename(w http.ResponseWriter, r *http.Request) {
	listID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid list id", ErrValidation).Error()})
		return
	}

	var req ListRenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: name is required", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	seq := query.ListRename(ctx, a.DB, listID, req.Name)
	renamed := false

	for _, err := range seq {
		if err != nil {
			log.Printf("list rename failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		renamed = true
	}

	if !renamed {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "list not found"})
		return
	}

	writeNoContent(w)
}

func (a *API) handleListReorder(w http.ResponseWriter, r *http.Request) {
	listID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid list id", ErrValidation).Error()})
		return
	}

	var req ListReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	seq := query.ListReorder(ctx, a.DB, listID, req.Position)
	reordered := false

	for _, err := range seq {
		if err != nil {
			log.Printf("list reorder failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		reordered = true
	}

	if !reordered {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "list not found"})
		return
	}

	writeNoContent(w)
}

func (a *API) handleCardCreate(w http.ResponseWriter, r *http.Request) {
	var req CardCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: title is required", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	card, err := query.CardCreate(ctx, a.DB, req.Title, req.Description, req.Position)
	if err != nil {
		log.Printf("card create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

		return
	}

	writeJSON(w, http.StatusCreated, card)
}

func (a *API) handleCardUpdate(w http.ResponseWriter, r *http.Request) {
	cardID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid card id", ErrValidation).Error()})
		return
	}

	var req CardUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: title is required", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	seq := query.CardUpdate(ctx, a.DB, cardID, req.Title, req.Description)
	updated := false

	for _, err := range seq {
		if err != nil {
			log.Printf("card update failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		updated = true
	}

	if !updated {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "card not found"})
		return
	}

	writeNoContent(w)
}

func (a *API) handleCardMove(w http.ResponseWriter, r *http.Request) {
	cardID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid card id", ErrValidation).Error()})
		return
	}

	var req CardMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.TargetListID == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: target_list_id is required", ErrValidation).Error()})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	seq := query.CardMove(ctx, a.DB, cardID, req.TargetListID, req.TargetPosition)
	moved := false

	for _, err := range seq {
		if err != nil {
			log.Printf("card move failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		moved = true
	}

	if !moved {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "card or target list not found"})
		return
	}

	writeNoContent(w)
}

func (a *API) handleCardReorder(w http.ResponseWriter, r *http.Request) {
	cardID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid card id", ErrValidation).Error()})
		return
	}

	var req CardReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	ctx := contextWithSystemDefaults(r.Context())

	seq := query.CardReorder(ctx, a.DB, cardID, req.Position)
	reordered := false

	for _, err := range seq {
		if err != nil {
			log.Printf("card reorder failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		reordered = true
	}

	if !reordered {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "card not found"})
		return
	}

	writeNoContent(w)
}

func (a *API) handleCardCommentList(w http.ResponseWriter, r *http.Request) {
	cardID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid card id", ErrValidation).Error()})
		return
	}

	// No need for existence check - foreign key constraint ensures card_id validity
	ctx := contextWithSystemDefaults(r.Context())
	seq := query.CardCommentList(ctx, a.DB, cardID)
	comments := make([]query.CardCommentListResult, 0)

	for item, err := range seq {
		if err != nil {
			log.Printf("card comment list failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

			return
		}

		if item != nil {
			comments = append(comments, *item)
		}
	}

	writeJSON(w, http.StatusOK, comments)
}

func (a *API) handleCardCommentCreate(w http.ResponseWriter, r *http.Request) {
	cardID, err := parsePathInt(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: invalid card id", ErrValidation).Error()})
		return
	}

	var req CardCommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Errorf("%w: text is required", ErrValidation).Error()})
		return
	}

	// No need for existence check - foreign key constraint will fail if card doesn't exist
	ctx := contextWithSystemDefaults(r.Context())

	comment, err := query.CardCommentCreate(ctx, a.DB, cardID, req.Text)
	if err != nil {
		// Check if this is a foreign key constraint violation (card not found)
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "card not found"})
			return
		}

		log.Printf("card comment create failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})

		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

func parsePathInt(r *http.Request, key string) (int, error) {
	val := r.PathValue(key)
	if val == "" {
		return 0, strconv.ErrSyntax
	}

	return strconv.Atoi(val)
}

func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func contextWithSystemDefaults(parent context.Context) context.Context {
	now := time.Now().UTC()
	ctx := snapsqlgo.WithSystemValue(parent, "created_at", now)
	ctx = snapsqlgo.WithSystemValue(ctx, "updated_at", now)

	return ctx
}

func createListsFromTemplatesWithTx(ctx context.Context, tx *sql.Tx, boardID int) ([]query.ListCreateResult, error) {
	// Use the ListCreate query which creates all lists from templates
	result, err := query.ListCreate(ctx, tx, boardID)
	if err != nil {
		return nil, err
	}

	// Return as a slice for consistency with the expected response format
	return []query.ListCreateResult{result}, nil
}
