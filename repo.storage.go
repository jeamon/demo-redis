package main

import "context"

type BookStorage interface {
	Add(ctx context.Context, id string, book Book) error
	GetOne(ctx context.Context, id string) (Book, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, book Book) (Book, error)
	GetAll(ctx context.Context) ([]Book, error)
}
