package main

import (
    "time"
    "os"
    "os/signal"
    "sync"

    "github.com/vyzigold/lokean/pkg/logs"

    "github.com/infrawatch/apputils/connector"
    "github.com/infrawatch/apputils/logging"
//    "github.com/infrawatch/apputils/config"
)

//spawnSignalHandler spawns goroutine which will wait for interruption signal(s)
// and end lokean in case any of the signal is received
func spawnSignalHandler(finish chan bool, logger *logging.Logger, watchedSignals ...os.Signal) {
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, watchedSignals...)
	go func() {
	signalLoop:
		for range interruptChannel {
			logger.Error("Stopping execution on caught signal")
			close(finish)
			break signalLoop
		}
	}()
}


func main() {
    // TODO: take the values from config file
    logger, err := logging.NewLogger(logging.DEBUG, "/dev/stderr")

    finish := make(chan bool)
    var wait sync.WaitGroup
    spawnSignalHandler(finish, logger, os.Interrupt)

    amqpClient := connector.NewAMQPServer(
        "amqp://127.0.0.1:5672/lokean/logs", //URL
        true, //debug
        1, //msgcount
        0, //prefetch
        "logs123") //name

    lokiClient, err := connector.NewLokiConnector(
        "http://localhost:3100", //URL
        2, //batchSize
        100 * time.Millisecond) //maxWaitTime

    if err != nil {
        // TODO: make the message more detailed once the config
        // file is read
        logger.Error("Couldn't create a loki client.")
        os.Exit(1)
    }

    lokiClient.Start()
    defer func() {
        lokiClient.Shutdown()
    }()

    logs.Run(amqpClient, lokiClient, logger, finish, &wait)

    wait.Wait()
}
