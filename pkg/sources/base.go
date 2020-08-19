package sources

import (
	"sync"

	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"
)

//LogSource interface for various sources Lokean understands
type LogSource interface {
	CreateLokiLog() (connector.LokiLog, error)
}

func contains(item string, slice []string) bool {
	for _, itm := range slice {
		if item == itm {
			return true
		}
	}
	return false
}

//Run is the main loop for receiving messages from the bus
func Run(receiver chan interface{}, sender chan interface{}, logger *logging.Logger, finish chan bool, wait *sync.WaitGroup) {
	wait.Add(1)
	go func() {
		logger.Debug("Starting log goroutine")
		defer func() {
			wait.Done()
			logger.Debug("Log goroutine finished")
		}()
	runLoop:
		for {
			select {
			case <-finish:
				break runLoop
			case rawMessage := <-receiver:
				logger.Metadata(map[string]interface{}{
					"message": rawMessage,
				})
				logger.Debug("Received message")

				var log *connector.LokiLog
				var err error
				switch msg := rawMessage.(type) {
				case connector.AMQP10Message:
					if contains("rsyslog", msg.Tags) {
						log, err = rsyslogMessageToLoki(msg.Body, logger)
					}
				case string:
					log, err = rsyslogMessageToLoki(msg, logger)
				default:
					logger.Error("Received unknown message type")
				}
				if err != nil {
					// error is noted in previous steps, so we just skip the sending here
					continue runLoop
				}
				if log != nil {
					sender <- *log
					logger.Debug("Log message sent")
				} else {
					logger.Metadata(map[string]interface{}{
						"message": rawMessage,
						"log":     log,
						"error":   err,
					})
					logger.Error("Unknown error during loki log creation")
				}

			}
		}
	}()
}
