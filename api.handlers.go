package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var EmptyData = struct{}{}

// Statistics holds app stats for ops.
type Statistics struct {
	version string
	called  uint64
	started time.Time
}

// Maintenance holds app maintenance mode infos.
type Maintenance struct {
	enabled atomic.Bool
	message string
	started time.Time
}

// APIHandler defines the API handler.
type APIHandler struct {
	logger      *zap.Logger
	config      *Config
	stats       *Statistics
	mode        *Maintenance
	bookService BookServiceProvider
}

// NewAPIHandler provides a new instance of APIHandler.
func NewAPIHandler(logger *zap.Logger, config *Config, stats *Statistics, bs BookServiceProvider) *APIHandler {
	m := &Maintenance{}
	m.enabled.Store(false)
	return &APIHandler{logger: logger, config: config, stats: stats, mode: m, bookService: bs}
}

// Index provides same details like `Status` handler by redirecting the request.
func (api *APIHandler) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.Redirect(w, r, "/status", http.StatusSeeOther)
}

// Status provides basics details about the application to the public users.
func (api *APIHandler) Status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"requestid": requestID,
			"status":    fmt.Sprintf("up & running since %.0f mins", time.Since(api.stats.started).Minutes()),
			"message":   "Hello. Books store api is available. Enjoy :)",
		},
	); err != nil {
		api.logger.Error("failed to send status response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// Maintenance handles request to enable or disable the maintenance mode of the service and respond
// to client requests with predefined message when the service is in maintenance mode.
// Enable the maintenance mode : /ops/maintenance?status=enable&msg=message-to-be-displayed-to-users
// Disable the maintenance mode: /ops/maintenance?status=disable
func (api *APIHandler) Maintenance(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var response map[string]interface{}
	var logger *zap.Logger

	q := r.URL.Query()
	mstatus := "show"
	if ps.ByName("status") != mstatus {
		mstatus = q.Get("status")
	}

	switch mstatus {
	case "enable":
		api.mode.message = q.Get("msg")
		api.mode.started = time.Now()
		api.mode.enabled.Store(true)
		response = map[string]interface{}{
			"requestid":           requestID,
			"maintenance.started": api.mode.started.Format(time.RFC1123),
			"maintenance.message": api.mode.message,
			"message":             "Maintenance mode enabled successfully.",
		}
		logger = api.logger.With(zap.String("request.id", requestID))

	case "disable":
		api.mode.enabled.Store(false)
		api.mode.started = time.Time{}
		api.mode.message = ""
		response = map[string]interface{}{
			"requestid": requestID,
			"message":   "Maintenance mode disabled successfully.",
		}
		logger = api.logger.With(zap.String("request.id", requestID))

	case "show":
		response = map[string]interface{}{
			"message": "service currently unvailable.",
			"reason":  api.mode.message,
			"since":   api.mode.started.UTC().Format(time.RFC1123),
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("failed to send maintenance response",
			zap.String("request.maintenance", mstatus),
			zap.Error(err),
		)
	}
}

// GetStatistics provides useful details about the application to the internal users.
func (api *APIHandler) GetStatistics(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"requestid": requestID,
			"version":   api.stats.version,
			"called":    atomic.LoadUint64(&api.stats.called),
			"started":   api.stats.started.Format(time.RFC1123),
			"uptime":    fmt.Sprintf("%.0f mins", time.Since(api.stats.started).Minutes()),
			"maintenance": map[string]interface{}{
				"enabled": api.mode.enabled.Load(),
				"started": api.mode.started,
				"message": api.mode.message,
			},
		},
	); err != nil {
		api.logger.Error("failed to send statistics response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// GetConfigs serves current in-use configurations/settings.
func (api *APIHandler) GetConfigs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"configs": api.config,
		},
	); err != nil {
		api.logger.Error("failed to send settings response", zap.String("request.id", requestID), zap.Error(err))
	}
}

func (api *APIHandler) CreateBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	book := Book{}
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	err := DecodeCreateOrUpdateBookRequestBody(r, &book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to create the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	err = ValidateCreateBookRequestBody(&book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to create the book", err)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	book.ID = GenerateID(BookIDPrefix)
	book.CreatedAt = time.Now().UTC().String()
	book.UpdatedAt = time.Now().UTC().String()

	err = api.bookService.Add(r.Context(), book.ID, book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to create the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	resp := GenericResponse(requestID, http.StatusCreated, "Book created successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}

func (api *APIHandler) GetAllBooks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	books, err := api.bookService.GetAll(r.Context())
	if err != nil {
		api.logger.Error("failed to get all books", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to get all books", books)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to get all books", zap.String("request.id", requestID))
	total := len(books)
	resp := GenericResponse(requestID, http.StatusOK, "All books fetched successfully.", &total, books)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.Error(err))
	}
}

func (api *APIHandler) GetOneBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	id := ps.ByName("id")
	book, err := api.bookService.GetOne(r.Context(), id)
	if err == ErrNotFoundBook {
		api.logger.Error("book does not exist", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to get book", zap.String("book.id", id), zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to create the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to get book", zap.String("book.id", id), zap.String("request.id", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book fetched successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}

func (api *APIHandler) DeleteOneBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	id := ps.ByName("id")
	book, err := api.bookService.GetOne(r.Context(), id)
	if err == ErrNotFoundBook {
		api.logger.Error("book does not exist", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to check if the book exist", zap.String("book.id", id), zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to check if the book exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	err = api.bookService.Delete(r.Context(), id)
	if err == ErrNotFoundBook {
		api.logger.Error("book does not exist", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to delete book", zap.String("book.id", id), zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to delete the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to delete book", zap.String("book.id", id), zap.String("request.id", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book deleted successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}

func (api *APIHandler) UpdateBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var book Book
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	err := DecodeCreateOrUpdateBookRequestBody(r, &book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to update the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	err = ValidateUpdateBookRequestBody(&book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to update the book", err)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	book, err = api.bookService.Update(r.Context(), book.ID, book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to update the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to update book", zap.String("book.id", book.ID), zap.String("request.id", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book updated successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}
