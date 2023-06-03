package main

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var EmptyData = struct{}{}

// Statistics holds app stats for ops.
type Statistics struct {
	version   string
	container bool
	runtime   string
	platform  string
	called    uint64
	started   time.Time
	status    map[int]uint64
	mu        *sync.RWMutex
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
	stats.status = make(map[int]uint64)
	stats.mu = &sync.RWMutex{}
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
		api.mode.started = time.Now().UTC()
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
		api.mode.started = time.Time{}.UTC()
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
			"since":   api.mode.started.Format(time.RFC1123),
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

// export goroutines to be used by expvar handler.
var goroutines = expvar.NewInt("goroutines")

// GetMemStats returns memory statistics with number of goroutines in json.
func GetMemStats(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	goroutines.Set(int64(runtime.NumGoroutine()))
	expvar.Handler().ServeHTTP(w, r)
}

// RunGC forces the run of the garbage collector asynchronously.
func (api *APIHandler) RunGC(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	go runtime.GC()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]string{
			"called": "go runtime.GC()",
		},
	); err != nil {
		api.logger.Error("failed to send run gc response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// FreeOSMemory forces the garbage collector to and tries to returns the memory
// back to the operating system in an asynchronous fashion.
func (api *APIHandler) FreeOSMemory(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	go debug.FreeOSMemory()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]string{
			"called": "go debug.FreeOSMemory()",
		},
	); err != nil {
		api.logger.Error("failed to send free os memory response", zap.String("request.id", requestID), zap.Error(err))
	}
}

// GetStatistics provides useful details about the application to the internal ops users.
// The stats returns by this handler do not contain the ops request which triggered that.
// That is why we remove 1 from the called field value in order to match the status stats.
func (api *APIHandler) GetStatistics(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	api.stats.mu.RLock()
	maintenanceModeStartedTime := api.mode.started.String()
	if api.mode.started == (time.Time{}.UTC()) {
		maintenanceModeStartedTime = ""
	}
	err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"requestid":     requestID,
			"app.version":   api.stats.version,
			"app.container": api.stats.container,
			"app.platform":  api.stats.platform,
			"go.version":    api.stats.runtime,
			"called":        atomic.LoadUint64(&api.stats.called) - 1,
			"started":       api.stats.started.Format(time.RFC1123),
			"uptime":        fmt.Sprintf("%.0f mins", time.Since(api.stats.started).Minutes()),
			"maintenance": map[string]interface{}{
				"enabled": api.mode.enabled.Load(),
				"started": maintenanceModeStartedTime,
				"message": api.mode.message,
			},
			"status": api.stats.status,
		},
	)
	api.stats.mu.RUnlock()
	if err != nil {
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

func (api *APIHandler) CreateBook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
	if err == ErrBookNotFound {
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
	if err == ErrBookNotFound {
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
	if err == ErrBookNotFound {
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

func (api *APIHandler) UpdateBook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
