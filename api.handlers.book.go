package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

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
			"status":    fmt.Sprintf("up & running since %.0f mins", api.clock.Now().Sub(api.stats.started).Minutes()),
			"message":   "Hello. Books store api is available. Enjoy :)",
		},
	); err != nil {
		api.logger.Error("failed to send status response", zap.String("request.id", requestID), zap.Error(err))
	}
}

func (api *APIHandler) CreateBook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	book := Book{}
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	err := DecodeCreateOrUpdateBookRequestBody(r, &book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to create the book", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	err = ValidateCreateBookRequestBody(&book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to create the book", err.Error())
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	book.ID = api.idsHandler.Generate(BookIDPrefix)
	book.CreatedAt = api.clock.Now().UTC().String()
	book.UpdatedAt = api.clock.Now().UTC().String()

	err = api.bookService.Add(r.Context(), book.ID, book)
	if err != nil {
		api.logger.Error("failed to create book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to create the book", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	resp := GenericResponse(requestID, http.StatusCreated, "Book created successfully.", nil, book)
	if err = WriteResponse(r.Context(), w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}

//nolint:bodyclose
func (api *APIHandler) GetAllBooks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	// this block could be moved into the TimeoutMiddleware and remove SetWriteDeadline and
	// ReadWriteDeadline methods from *CustomResponseWriter object because that middleware
	// is called before the stats middleware which wraps the native ResponseWriter.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Now().Add(api.config.Server.LongRequestWriteTimeout)); err != nil {
		api.logger.Error("http: failed to update the write deadline", zap.String("request.id", requestID), zap.Error(err))
	}

	books, err := api.bookService.GetAll(r.Context())
	if err != nil {
		api.logger.Error("failed to get all books", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to get all books", books)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to get all books", zap.String("request.id", requestID))
	total := len(books)
	resp := GenericResponse(requestID, http.StatusOK, "All books fetched successfully.", &total, books)
	if err = WriteResponse(r.Context(), w, resp); err != nil {
		api.logger.Error("failed to send response", zap.Error(err))
	}
}

func (api *APIHandler) GetOneBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	id := ps.ByName("id")
	if ok := api.idsHandler.IsValid(id, BookIDPrefix); !ok {
		api.logger.Error("book id provided is not valid", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "book id provided is not valid", Book{})
		if err := WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	book, err := api.bookService.GetOne(r.Context(), id)
	if err == ErrBookNotFound {
		api.logger.Error("book does not exist", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to get book", zap.String("book.id", id), zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to create the book", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to get book", zap.String("book.id", id), zap.String("request.id", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book fetched successfully.", nil, book)
	if err = WriteResponse(r.Context(), w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}

func (api *APIHandler) DeleteOneBook(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	requestID := GetValueFromContext(r.Context(), ContextRequestID)
	id := ps.ByName("id")
	if ok := api.idsHandler.IsValid(id, BookIDPrefix); !ok {
		api.logger.Error("book id provided is not valid", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "book id provided is not valid", Book{})
		if err := WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	book, err := api.bookService.GetOne(r.Context(), id)
	if err == ErrBookNotFound {
		api.logger.Error("book does not exist", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to check if the book exist", zap.String("book.id", id), zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to check if the book exist", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	err = api.bookService.Delete(r.Context(), id)
	if err == ErrBookNotFound {
		api.logger.Error("book does not exist", zap.String("book.id", id), zap.String("request.id", requestID))
		errResp := NewAPIError(requestID, http.StatusNotFound, "book does not exist", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	if err != nil {
		api.logger.Error("failed to delete book", zap.String("book.id", id), zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to delete the book", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to delete book", zap.String("book.id", id), zap.String("request.id", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book deleted successfully.", nil, book)
	if err = WriteResponse(r.Context(), w, resp); err != nil {
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
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	err = ValidateUpdateBookRequestBody(&book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusBadRequest, "failed to update the book", err.Error())
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}

	book, err = api.bookService.Update(r.Context(), book.ID, book)
	if err != nil {
		api.logger.Error("failed to update book", zap.String("request.id", requestID), zap.Error(err))
		errResp := NewAPIError(requestID, http.StatusInternalServerError, "failed to update the book", book)
		if err = WriteErrorResponse(r.Context(), w, errResp); err != nil {
			api.logger.Error("failed to send error response", zap.String("request.id", requestID), zap.Error(err))
		}
		return
	}
	api.logger.Info("success to update book", zap.String("book.id", book.ID), zap.String("request.id", requestID))
	resp := GenericResponse(requestID, http.StatusOK, "Book updated successfully.", nil, book)
	if err = WriteResponse(r.Context(), w, resp); err != nil {
		api.logger.Error("failed to send response", zap.String("request.id", requestID), zap.Error(err))
	}
}
