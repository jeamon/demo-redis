package main

import (
	"log"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("app failed: %v", err)
	}
	err = app.Run()
	if err != nil {
		log.Printf("app exited: %v", err)
	}
}
