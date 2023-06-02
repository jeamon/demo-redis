package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
	"go.uber.org/zap"
)

type boltBookStorage struct {
	logger *zap.Logger
	client *bolt.DB
	config *BoltDBConfig
}

// GetBoltClient setup the database and the bucket then provides a ready to use client.
func GetBoltDBClient(config *Config) (*bolt.DB, error) {
	db, err := bolt.Open(config.BoltDB.FilePath, 0o600, &bolt.Options{Timeout: config.BoltDB.Timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to open the database, %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, errB := tx.CreateBucketIfNotExists([]byte(config.BoltDB.BucketName)); errB != nil {
			return fmt.Errorf("failed to create %s bucket: %v", config.BoltDB.BucketName, errB)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up bucket: %v", err)
	}
	return db, nil
}

// NewBoltBookStorage provides an instance of bolt-based book storage.
func NewBoltBookStorage(logger *zap.Logger, boltConfig *BoltDBConfig, client *bolt.DB) BookStorage {
	return &boltBookStorage{
		logger: logger,
		client: client,
		config: boltConfig,
	}
}

// Close shuts down the bolt-based book storage.
func (bs *boltBookStorage) Close() error {
	return bs.client.Close()
}

// Add inserts a new book record into boltdb store.
func (bs *boltBookStorage) Add(_ context.Context, id string, book Book) error {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return err
	}
	err = bs.client.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bs.config.BucketName)).Put([]byte(id), bookBytes)
	})
	return err
}

// GetOne retrieves a book record based on its ID from boltdb store.
func (bs *boltBookStorage) GetOne(_ context.Context, id string) (Book, error) {
	var book Book
	// initialize a readable transaction.
	tx, err := bs.client.Begin(false)
	if err != nil {
		return book, err
	}
	defer tx.Rollback()

	result := tx.Bucket([]byte(bs.config.BucketName)).Get([]byte(id))
	if result == nil {
		return book, ErrNotFoundBook
	}
	err = json.Unmarshal(result, &book)
	return book, err
}

// Delete removes a book record based on its ID from boltdb store.
func (bs *boltBookStorage) Delete(_ context.Context, id string) error {
	return bs.client.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bs.config.BucketName)).Delete([]byte(id))
	})
}

// Update replaces existing book record data or inserts a new book if does not exist.
func (bs *boltBookStorage) Update(_ context.Context, id string, book Book) (Book, error) {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return book, err
	}
	err = bs.client.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bs.config.BucketName)).Put([]byte(id), bookBytes)
	})
	return book, err
}

// GetAll retrieves a list of all books stored in the bolt database.
func (bs *boltBookStorage) GetAll(_ context.Context) ([]Book, error) {
	tx, err := bs.client.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create a cursor on the books' bucket.
	c := tx.Bucket([]byte(bs.config.BucketName)).Cursor()

	books := []Book{}
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var book Book
		if err = json.Unmarshal(v, &book); err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	return books, nil
}
