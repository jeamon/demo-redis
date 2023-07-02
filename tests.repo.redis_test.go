package main

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/go-redis/redis/v9"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func startRedisDockerContainer(t *testing.T) (string, func()) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Failed to start Dockertest: %+v", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		t.Fatalf("Could not connect to Docker: %+v", err)
	}

	resource, err := pool.Run("redis", "7.0.10-alpine", nil)
	if err != nil {
		t.Fatalf("Failed to start redis: %+v", err)
	}

	// build address the container is listening on
	addr := net.JoinHostPort("localhost", resource.GetPort("6379/tcp"))

	// ensure to wait for the container to be ready
	err = pool.Retry(func() error {
		var e error
		client := redis.NewClient(&redis.Options{Addr: addr})
		defer client.Close()

		e = client.Ping(context.Background()).Err()
		return e
	})

	if err != nil {
		t.Fatalf("Failed to ping Redis: %+v", err)
	}

	destroyFunc := func() {
		if err := pool.Purge(resource); err != nil {
			t.Logf("Failed to purge resource: %+v", err)
		}
	}

	return addr, destroyFunc
}

func TestRedisStore(t *testing.T) {
	addr, destroyFunc := startRedisDockerContainer(t)
	defer destroyFunc()
	rs := NewRedisBookStorage(zap.NewNop(), redis.NewClient(&redis.Options{Addr: addr}))
	testBook0ID, testBook1ID := "b:0", "b:1"
	testBook := Book{
		ID:          testBook0ID,
		Title:       "Redis test book title",
		Description: "Redis test book desc",
		Author:      "Jerome Amon",
		Price:       "10$",
		CreatedAt:   "2023-07-01 20:19:10.7604632 +0000 UTC",
		UpdatedAt:   "2023-07-01 20:19:10.7604632 +0000 UTC",
	}

	t.Run("Add Book", func(t *testing.T) {
		// ensures we can insert new book record.
		err := rs.Add(context.Background(), testBook0ID, testBook)
		assert.NoError(t, err)
	})

	t.Run("Get Existent Book", func(t *testing.T) {
		// ensures we can fetch specific book.
		book, err := rs.GetOne(context.Background(), testBook0ID)
		assert.NoError(t, err)
		if !reflect.DeepEqual(testBook, book) {
			t.Errorf("Got %v but Expected %v.", book, testBook)
		}
	})

	t.Run("Get NonExistent Book", func(t *testing.T) {
		// ensures fetching non-existent book fails.
		book, err := rs.GetOne(context.Background(), testBook1ID)
		assert.Equal(t, ErrBookNotFound, err)
		assert.Equal(t, Book{}, book)
	})

	t.Run("Delete Existent Book", func(t *testing.T) {
		// ensures deleting existent book succeed.
		err := rs.Delete(context.Background(), testBook0ID)
		assert.NoError(t, err)
		book, err := rs.GetOne(context.Background(), testBook0ID)
		assert.Equal(t, ErrBookNotFound, err)
		assert.Equal(t, Book{}, book)
	})

	t.Run("Delete NonExistent Book", func(t *testing.T) {
		// ensures deleting non existent book returns an error.
		err := rs.Delete(context.Background(), testBook1ID)
		assert.Equal(t, ErrBookNotFound, err)
	})

	t.Run("Update NonExistent Book", func(t *testing.T) {
		// ensures updating non-existing book create that book.
		book, err := rs.Update(context.Background(), testBook0ID, testBook)
		assert.NoError(t, err)
		if !reflect.DeepEqual(testBook, book) {
			t.Errorf("Got %v but Expected %v.", book, testBook)
		}
		book, err = rs.GetOne(context.Background(), testBook0ID)
		assert.NoError(t, err)
		if !reflect.DeepEqual(testBook, book) {
			t.Errorf("Got %v but Expected %v.", book, testBook)
		}
	})

	t.Run("Update Existent Book", func(t *testing.T) {
		// ensures we can update an existent book record.
		testBook.Price = "20$"
		book, err := rs.Update(context.Background(), testBook0ID, testBook)
		assert.NoError(t, err)
		if !reflect.DeepEqual(testBook, book) {
			t.Errorf("Got %v but Expected %v.", book, testBook)
		}
		book, err = rs.GetOne(context.Background(), testBook0ID)
		assert.NoError(t, err)
		assert.Equal(t, testBook.Price, book.Price)
	})

	t.Run("Get All Books", func(t *testing.T) {
		// ensures we get exact number of stored books.
		err := rs.Add(context.Background(), testBook1ID, testBook)
		assert.NoError(t, err)
		books, err := rs.GetAll(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 2, len(books))
	})
}
