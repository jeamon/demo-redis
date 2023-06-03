package main

import (
	"context"
	"encoding/json"
	"fmt"

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
		DialTimeout:  config.Redis.DialTimeout,
		ReadTimeout:  config.Redis.ReadTimeout,
		WriteTimeout: config.Redis.WriteTimeout,
		PoolSize:     config.Redis.PoolSize,
		PoolTimeout:  config.Redis.PoolTimeout,
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
func (rs *redisBookStorage) Add(ctx context.Context, id string, book Book) error {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return err
	}
	return rs.client.HSet(ctx, HBooks, id, bookBytes).Err()
}

// GetOne retrieves a book record based on its ID.
func (rs *redisBookStorage) GetOne(ctx context.Context, id string) (Book, error) {
	var book Book
	bookJSONString, err := rs.client.HGet(ctx, HBooks, id).Result()
	if err == redis.Nil {
		return book, ErrBookNotFound
	}
	if err != nil {
		return book, err
	}
	err = json.Unmarshal([]byte(bookJSONString), &book)
	return book, err
}

// Delete removes a book record based on its ID.
func (rs *redisBookStorage) Delete(ctx context.Context, id string) error {
	err := rs.client.HDel(ctx, HBooks, id).Err()
	if err == redis.Nil {
		return ErrBookNotFound
	}
	return err
}

// Update replaces existing book record data or inserts a new book if does not exist.
func (rs *redisBookStorage) Update(ctx context.Context, id string, book Book) (Book, error) {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return book, err
	}
	err = rs.client.HSet(ctx, HBooks, id, bookBytes).Err()
	return book, err
}

// GetAll retrieves a list of all books stored in the redis database.
func (rs *redisBookStorage) GetAll(ctx context.Context) ([]Book, error) {
	mapBooks, err := rs.client.HVals(ctx, HBooks).Result()
	if err != nil {
		return nil, err
	}
	books := []Book{}
	for _, bookJSONString := range mapBooks {
		var book Book
		if err = json.Unmarshal([]byte(bookJSONString), &book); err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, nil
}
