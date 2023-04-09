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

type Statistics struct {
	version string
	called  uint64
	started time.Time
}

// APIHandler defines the API handler.
type APIHandler struct {
	logger      *zap.Logger
	config      *Config
	stats       *Statistics
	bookService BookServiceProvider
}

// NewAPIHandler provides a new instance of APIHandler.
func NewAPIHandler(logger *zap.Logger, config *Config, stats *Statistics, bs BookServiceProvider) *APIHandler {
	return &APIHandler{logger: logger, config: config, stats: stats, bookService: bs}
}

func (api *APIHandler) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetRequestIDFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"called":  atomic.LoadUint64(&api.stats.called),
			"status":  fmt.Sprintf("up & running since %.0f mins", time.Since(api.stats.started).Minutes()),
			"message": "Hello. Books store api is available. Enjoy :)",
		},
	); err != nil {
		api.logger.Error("failed to send index response", zap.String("requestid", requestID), zap.Error(err))
	}
}

// GetConfigs serves current in-use configurations/settings.
func (api *APIHandler) GetConfigs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetRequestIDFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(
		map[string]interface{}{
			"configs": api.config,
		},
	); err != nil {
		api.logger.Error("failed to send settings response", zap.String("requestid", requestID), zap.Error(err))
	}
}

func (api *APIHandler) CreateBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	book := Book{}
	requestID := GetRequestIDFromContext(r.Context())
	err := DecodeCreateOrUpdateBookRequestBody(r, &book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to create the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}

	err = ValidateCreateBookRequestBody(&book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to create the book", err)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}

	book.ID = GenerateBookID()
	book.CreatedAt = time.Now().UTC().String()
	book.UpdatedAt = time.Now().UTC().String()

	err = api.bookService.Add(r.Context(), book.ID, book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to create the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to create book", zap.String("requestid", requestID), zap.String("requestid", requestID))
	resp := GenericResponse(requestID, http.StatusCreated, "Book created successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("requestid", requestID), zap.Error(err))
	}
}

func (api *APIHandler) GetAllBooks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetRequestIDFromContext(r.Context())
	books, err := api.bookService.GetAll(r.Context())
	if err != nil {
		api.logger.Error("failed to get all books", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to get all books", books)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to get all books", zap.String("requestid", requestID))
	total := len(books)
	resp := GenericResponse(requestID, http.StatusOK, "All books fetched successfully.", &total, books)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.Error(err))
	}
}

func (api *APIHandler) GetOneBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetRequestIDFromContext(r.Context())
	id := ps.ByName("id")
	book, err := api.bookService.GetOne(r.Context(), id)
	if err == ErrNotFoundBook {
		api.logger.Error("book does not exist", zap.String("id", id), zap.String("requestid", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to get book", zap.String("id", id), zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to create the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to get book", zap.String("id", id), zap.String("requestid", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book fetched successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("requestid", requestID), zap.Error(err))
	}
}

func (api *APIHandler) DeleteOneBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetRequestIDFromContext(r.Context())
	id := ps.ByName("id")
	book, err := api.bookService.GetOne(r.Context(), id)
	if err == ErrNotFoundBook {
		api.logger.Error("book does not exist", zap.String("id", id), zap.String("requestid", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to check if the book exist", zap.String("id", id), zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to check if the book exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}

	err = api.bookService.Delete(r.Context(), id)
	if err == ErrNotFoundBook {
		api.logger.Error("book does not exist", zap.String("id", id), zap.String("requestid", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to delete book", zap.String("id", id), zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to delete the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to delete book", zap.String("id", id), zap.String("requestid", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book deleted successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("requestid", requestID), zap.Error(err))
	}
}

func (api *APIHandler) UpdateBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var book Book
	requestID := GetRequestIDFromContext(r.Context())
	err := DecodeCreateOrUpdateBookRequestBody(r, &book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to update the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}

	err = ValidateUpdateBookRequestBody(&book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to update the book", err)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}

	book, err = api.bookService.Update(r.Context(), book.ID, book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("requestid", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to update the book", book)
		if err = WriteErrorResponse(w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("requestid", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to update book", zap.String("requestid", requestID), zap.String("requestid", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book updated successfully.", nil, book)
	if err = WriteResponse(w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("requestid", requestID), zap.Error(err))
	}
}
