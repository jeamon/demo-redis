package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Predefinied Queue IDs.
const (
	CreateQueue = "creation"
	UpdateQueue = "updating"
	DeleteQueue = "deletion"
)

// Ensure *Queue implements Queuer.
var _ Queuer = (*redisQueue)(nil)

// Queuer describes a queue.
type Queuer interface {
	Push(ctx context.Context, qid string, book Book) error
	Pop(ctx context.Context, qids ...string) (string, Book, error)
}

// redisQueue represents a queue which implements the Queuer interface.
type redisQueue struct {
	client *redis.Client
}

func NewRedisQueue(client *redis.Client) Queuer {
	return &redisQueue{client: client}
}

// Push enqueues a book onto the queue identified by qid.
func (q *redisQueue) Push(ctx context.Context, qid string, book Book) error {
	bookBytes, err := json.Marshal(book)
	if err != nil {
		return err
	}
	return q.client.RPush(ctx, qid, bookBytes).Err()
}

// Pop returns the first dequeued book from the list of queue ids.
func (q *redisQueue) Pop(ctx context.Context, qids ...string) (string, Book, error) {
	var book Book
	var qid string
	infos, err := q.client.BLPop(ctx, 0*time.Second, qids...).Result()
	if err != nil {
		return qid, book, err
	}

	if err = json.Unmarshal([]byte(infos[1]), &book); err != nil {
		return qid, book, err
	}
	qid = infos[0]
	return qid, book, nil
}
