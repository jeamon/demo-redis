package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"go.uber.org/zap"
)

const HBooks string = "books"

type redisBookStorage struct {
	logger *zap.Logger
	client *redis.Client
}

// NewRedisBookStorage provides an instance of redis-based book storage.
func NewRedisBookStorage(logger *zap.Logger, client *redis.Client) BookStorage {
	return &redisBookStorage{
		logger: logger,
		client: client,
	}
}

// GetRedisClient provides a ready to use redis client.
func GetRedisClient(config *Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port),
		DialTimeout:  time.Duration(config.Redis.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Redis.WriteTimeout) * time.Second,
		PoolSize:     config.Redis.PoolSize,
		PoolTimeout:  time.Duration(config.Redis.PoolTimeout) * time.Second,
		Password:     config.Redis.Password,
		Username:     config.Redis.Username,
		DB:           config.Redis.DatabaseIndex,
	})

	// test connection.
	if pong, err := client.Ping(context.Background()).Result(); pong != "PONG" || err != nil {
		return client, fmt.Errorf("test connection failed: %v", err)
	}
	return client, nil
}

// Add inserts a new book record.
func (r *redisBookStorage) Add(ctx context.Context, id string, book Book) error {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return err
	}
	return r.client.HSet(ctx, HBooks, id, bookBytes).Err()
}

// GetOne retrieves a book record based on its ID.
func (r *redisBookStorage) GetOne(ctx context.Context, id string) (Book, error) {
	var book Book
	bookJSONString, err := r.client.HGet(ctx, HBooks, id).Result()
	if err == redis.Nil {
		return book, ErrNotFoundBook
	}
	if err != nil {
		return book, err
	}
	err = json.Unmarshal([]byte(bookJSONString), &book)
	return book, err
}

// Delete removes a book record based on its ID.
func (r *redisBookStorage) Delete(ctx context.Context, id string) error {
	err := r.client.HDel(ctx, HBooks, id).Err()
	if err == redis.Nil {
		return ErrNotFoundBook
	}
	return err
}

// Update replaces existing book record data or inserts a new book if does not exist.
func (r *redisBookStorage) Update(ctx context.Context, id string, book Book) (Book, error) {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return book, err
	}
	err = r.client.HSet(ctx, HBooks, id, bookBytes).Err()
	return book, err
}

// GetAll replaces existing book record data or inserts a new book if does not exist.
func (r *redisBookStorage) GetAll(ctx context.Context) ([]Book, error) {
	mapBooks, err := r.client.HVals(ctx, HBooks).Result()
	if err != nil {
		return []Book{}, err
	}
	books := []Book{}
	for _, bookJSONString := range mapBooks {
		var book Book
		if err = json.Unmarshal([]byte(bookJSONString), &book); err != nil {
			r.logger.Error("failed to unmarshall details of book", zap.String("requestid", GetRequestIDFromContext(ctx)), zap.Error(err))
			continue
		}
		books = append(books, book)
	}
	return books, nil
}
