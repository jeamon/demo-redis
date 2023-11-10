package main

import (
	"context"
	"time"
)

// This file contains mocks definitions needed to perform unit tests.

type MockBookStorage struct {
	AddFunc       func(ctx context.Context, id string, book Book) error
	GetOneFunc    func(ctx context.Context, id string) (Book, error)
	DeleteFunc    func(ctx context.Context, id string) error
	UpdateFunc    func(ctx context.Context, id string, book Book) (Book, error)
	GetAllFunc    func(ctx context.Context) ([]Book, error)
	DeleteAllFunc func(ctx context.Context) error
}

// Add mocks the behavior of book creation by the repository.
func (m *MockBookStorage) Add(ctx context.Context, id string, book Book) error {
	return m.AddFunc(ctx, id, book)
}

// GetOne mocks the behavior of retrieving a book by the repository.
func (m *MockBookStorage) GetOne(ctx context.Context, id string) (Book, error) {
	return m.GetOneFunc(ctx, id)
}

// Delete mocks the behavior of deleting a book by the repository.
func (m *MockBookStorage) Delete(ctx context.Context, id string) error {
	return m.DeleteFunc(ctx, id)
}

// Update mocks the behavior of updating a book by the repository.
func (m *MockBookStorage) Update(ctx context.Context, id string, book Book) (Book, error) {
	return m.UpdateFunc(ctx, id, book)
}

// GetAll mocks the behavior of retrieving all books by the repository.
func (m *MockBookStorage) GetAll(ctx context.Context) ([]Book, error) {
	return m.GetAllFunc(ctx)
}

// DeleteAll mocks the behavior of deleting all books by the repository.
func (m *MockBookStorage) DeleteAll(ctx context.Context) error {
	return m.DeleteAllFunc(ctx)
}

// MockClocker implements a fake Clocker.
type MockClocker struct {
	MockNow time.Time
}

// NewMockClocker returns a mocked instance with fixed time.
func NewMockClocker() *MockClocker {
	return &MockClocker{time.Date(2023, 0o7, 0o2, 0o0, 0o0, 0o0, 0o00000000, time.UTC)}
}

// Now returns an already defined time to be used as mock. This
// equals to `Sun, 02 Jul 2023 00:00:00 UTC` in time.RFC1123 format.
// equals to `2023-07-02 00:00:00 +0000 UTC` in String format.
func (mck *MockClocker) Now() time.Time {
	return mck.MockNow
}

// MockUIDHandler implements a fake UIDHandler.
type MockUIDHandler struct {
	MockedUID string
	Valid     bool
}

// NewMockUIDHandler returns a mocked instance with predictable id.
func NewMockUIDHandler(id string, valid bool) *MockUIDHandler {
	return &MockUIDHandler{MockedUID: id, Valid: valid}
}

// Generate constructs a predictable id to be used as mock.
func (muid *MockUIDHandler) Generate(prefix string) string {
	return prefix + ":" + muid.MockedUID
}

// IsValid mocks IsValid behavior by providing configured status.
func (muid *MockUIDHandler) IsValid(_, _ string) bool {
	return muid.Valid
}

type MockQueuer struct {
	PushFunc func(ctx context.Context, qid string, book Book) error
	PopFunc  func(ctx context.Context, qids ...string) (string, Book, error)
}

// Push mocks the behavior of book enqueuing into the queue.
func (m *MockQueuer) Push(ctx context.Context, qid string, book Book) error {
	return m.PushFunc(ctx, qid, book)
}

// Pop mocks the behavior of deuqueing a book from the queue.
func (m *MockQueuer) Pop(ctx context.Context, qids ...string) (string, Book, error) {
	return m.PopFunc(ctx, qids...)
}

type MockConsumer struct {
	ConsumeFunc func(ctx context.Context, qids ...string)
}

func (m *MockConsumer) Consume(ctx context.Context, qids ...string) {
	m.ConsumeFunc(ctx, qids...)
}
