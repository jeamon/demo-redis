package main

import (
	"log"
)

var (
	GitCommit string
	GitTag    string
	BuildTime string
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal("application failed to initialized: ", err)
	}
	err = app.Run()
	if err != nil {
		log.Fatal("application exited. check logs for more details.", err)
	}
}
