package main

import (
	"log"
	"time"
)

func main() {
	app, err := NewApp(time.Now())
	if err != nil {
		log.Fatalf("app failed: %v", err)
	}
	err = app.Run()
	if err != nil {
		log.Printf("app exited: %v", err)
	}
}
