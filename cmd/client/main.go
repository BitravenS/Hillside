package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"hillside/internal/client"
)

func main() {
	logFile, err := os.OpenFile("panic.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	run()
}

func run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic: %v\nStack trace:\n%s", r, debug.Stack())
			fmt.Println("A fatal error occurred. Please check panic.log for details.")
			os.Exit(2)
		}
	}()

	var logPort int
	flag.IntVar(&logPort, "logport", 4567, "Port for remote logger")
	flag.Parse()
	client.StartClientApp(logPort)
}
