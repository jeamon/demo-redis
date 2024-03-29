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

// Ensure concrete type boltBookStorage satisfies BookStorage interface.
func TestBoltBookStorageImplementsBookStorageInterface(t *testing.T) {
	var i interface{} = new(boltBookStorage)
	if _, ok := i.(BookStorage); !ok {
		t.Fatalf("expected %T to implement BookStorage", i)
	}
}

// Ensure bolt store can insert a new book.
func TestBoltStore_AddBook(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBookID := "b:0"

	// Create a new book.
	b := Book{ID: testBookID, Title: "Bolt test book title"}
	err = bs.Add(context.TODO(), testBookID, b)
	require.NoError(t, err)

	// Verify book can be retrieved.
	book, err := bs.GetOne(context.TODO(), testBookID)
	require.NoError(t, err)
	assert.Equal(t, testBookID, book.ID)
	assert.Equal(t, "Bolt test book title", book.Title)
}

// Ensure bolt store returns exact book details if exist.
func TestBoltStore_GetOneBook_FoundBook(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBookID := "b:0"

	// Create a new book.
	b := Book{
		ID:          testBookID,
		Title:       "Bolt test book title",
		Description: "Bolt test book desc",
		Author:      "Jerome Amon",
		Price:       "10$",
		CreatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
		UpdatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
	}
	err = bs.Add(context.TODO(), testBookID, b)
	require.NoError(t, err)

	// Verify book does exist.
	book, err := bs.GetOne(context.TODO(), testBookID)
	require.NoError(t, err)
	assert.Equal(t, b, book)
}

// Ensure bolt store returns an error if book does not exist.
func TestBoltStore_GetOneBook_ErrBookNotFound(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBookID := "b:0"

	// Create a new book.
	b := Book{ID: testBookID, Title: "Bolt test book title"}
	err = bs.Add(context.TODO(), testBookID, b)
	require.NoError(t, err)

	// Verify another book does not exist.
	book, err := bs.GetOne(context.TODO(), "b:1")
	assert.Equal(t, ErrBookNotFound, err)
	assert.Equal(t, Book{}, book)
}

// Ensure bolt store can remove a book.
func TestBoltStore_DeleteBook(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBookID := "b:0"

	// Create a new book.
	b := Book{
		ID:          testBookID,
		Title:       "Bolt test book title",
		Description: "Bolt test book desc",
		Author:      "Jerome Amon",
		Price:       "10$",
		CreatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
		UpdatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
	}
	err = bs.Add(context.TODO(), testBookID, b)
	require.NoError(t, err)

	// Delete the book.
	err = bs.Delete(context.TODO(), testBookID)
	require.NoError(t, err)

	// Verify book does not exist.
	book, err := bs.GetOne(context.TODO(), testBookID)
	assert.Equal(t, ErrBookNotFound, err)
	assert.Equal(t, Book{}, book)
}

// Ensure bolt store can retrieve multiple books.
func TestBoltStore_GetAllBooks(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBook0ID := "b:0"
	testBook1ID := "b:1"

	// Create some new books.
	b0 := Book{ID: testBook0ID, Title: "Bolt test book 0 title"}
	err = bs.Add(context.TODO(), testBook0ID, b0)
	require.NoError(t, err)
	b1 := Book{ID: testBook1ID, Title: "Bolt test book 1 title"}
	err = bs.Add(context.TODO(), testBook1ID, b1)
	require.NoError(t, err)

	// Verify books can be retrieved.
	books, err := bs.GetAll(context.TODO())
	require.NoError(t, err)
	assert.ElementsMatch(t, books, []Book{b0, b1})
}

// Ensure bolt store can update an existing book details.
func TestBoltStore_UpdateBook_ExistingBook(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBookID := "b:0"

	// Create a new book.
	b := Book{
		ID:          testBookID,
		Title:       "Bolt test book title",
		Description: "Bolt test book desc",
		Author:      "Jerome Amon",
		Price:       "10$",
		CreatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
		UpdatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
	}
	err = bs.Add(context.TODO(), testBookID, b)
	require.NoError(t, err)

	// Modify existing book details and update.
	newBook := b
	newBook.Title = "Bolt test book new title"
	newBook.Price = "20$"
	newBook.UpdatedAt = time.Now().UTC().String()
	book, err := bs.Update(context.TODO(), testBookID, newBook)
	require.NoError(t, err)
	assert.Equal(t, book, newBook)

	// Check if book was updated.
	book, err = bs.GetOne(context.TODO(), testBookID)
	require.NoError(t, err)
	assert.Equal(t, book, newBook)
}

// Ensure bolt store inserts book during update operation if book
// does not exist.
func TestBoltStore_UpdateBook_NotExistingBook(t *testing.T) {
	bs, err := newTestBoltStore()
	require.NoError(t, err, "failed in creating a test bolt store")
	defer func() {
		err = bs.closeTestBoltStore()
		assert.NoError(t, err)
	}()

	testBookID := "b:0"

	// Use update operation to add book.
	b := Book{
		ID:          testBookID,
		Title:       "Bolt test book title",
		Description: "Bolt test book desc",
		Author:      "Jerome Amon",
		Price:       "10$",
		CreatedAt:   "2023-04-26 21:42:10.7604632 +0000 UTC",
		UpdatedAt:   time.Now().UTC().String(),
	}

	// Modify existing book details and update.
	book, err := bs.Update(context.TODO(), testBookID, b)
	require.NoError(t, err)
	assert.Equal(t, b, book)

	// Check if book was added.
	book, err = bs.GetOne(context.TODO(), testBookID)
	require.NoError(t, err)
	assert.Equal(t, b, book)
}
