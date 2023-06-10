package main

import "context"

// This file contains mocks definitions needed to perform unit tests.

type MockBookStorage struct {
	AddFunc    func(ctx context.Context, id string, book Book) error
	GetOneFunc func(ctx context.Context, id string) (Book, error)
	DeleteFunc func(ctx context.Context, id string) error
	UpdateFunc func(ctx context.Context, id string, book Book) (Book, error)
	GetAllFunc func(ctx context.Context) ([]Book, error)
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
