package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// NewStore returns a new instance of Store in a temporary path.
func newTestBoltStore() (*boltBookStorage, error) {
	f, err := os.CreateTemp("", "tmp.bolt.db-")
	if err != nil {
		return nil, err
	}
	f.Close()
	testConfig := &Config{
		BoltDB: BoltDBConfig{
			FilePath:   f.Name(),
			Timeout:    5 * time.Second,
			BucketName: "test.books",
		},
	}

	client, err := GetBoltDBClient(testConfig)

	return &boltBookStorage{
		logger: zap.NewNop(),
		client: client,
		config: &testConfig.BoltDB,
	}, err
}

// Close closes the temporary bolt store and removes the underlying data file.
func (bs *boltBookStorage) closeTestBoltStore() error {
	defer os.Remove(bs.config.FilePath)
	return bs.Close()
}

// Ensure bolt store can insert a new book.
func TestBoltStore_AddBook(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer bs.closeTestBoltStore()
	testBookID := "b:0"

	// Create a new book.
	b := Book{ID: testBookID, Title: "Bolt test book title"}
	err = bs.Add(context.TODO(), testBookID, b)
	assert.NoError(t, err)

	// Verify book can be retrieved.
	book, err := bs.GetOne(context.TODO(), testBookID)
	assert.NoError(t, err)
	assert.Equal(t, testBookID, book.ID)
	assert.Equal(t, "Bolt test book title", book.Title)
}
