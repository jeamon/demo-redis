package main

import (
	"log"
)

var (
	gitCommit string
	gitTag    string
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal("application failed to initialized: ", err)
	}
	err = app.Run()
	if err != nil {
		log.Println("application exited. check logs for more details.", err)
	}
}
