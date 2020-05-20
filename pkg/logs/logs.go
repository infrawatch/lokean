package logs

import (
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/connector"

	"encoding/json"
	"sync"
	"time"
	"fmt"
)

type Log struct {
	Level string `json:"level"`
	Timestamp int `json:"timestamp"`
	Source string `json:"source"`
	LogMessage string `json:"message"`
}

func createLokiLog(rawMessage interface{}, logger *logging.Logger) (connector.LokiLog, error) {
	switch msg := rawMessage.(type) {
	case connector.AMQP10Message:
		message := msg.Body
		logger.Debug("Received the folowing log message:")
		logger.Debug(message)
		var log Log
		err := json.Unmarshal([]byte(message), &log)
		if err != nil {
			return connector.LokiLog{}, err
		}
		labels := make(map[string]string)
		labels["source"] = log.Source
		logMessage := fmt.Sprintf("[%s] %s", log.Level, log.LogMessage)
		return connector.LokiLog {
			Labels: labels,
			LogMessage: logMessage,
			Timestamp: time.Duration(log.Timestamp) * time.Millisecond,
		}, nil
	default:
		return connector.LokiLog{}, fmt.Errorf("Received unknown message type")
	}
}

func Run(receiver chan interface{}, sender chan interface{}, logger *logging.Logger, finish chan bool, wait *sync.WaitGroup) {
	wait.Add(1)
	go func() {
		logger.Debug("Starting log goroutine")
		defer func () {
			wait.Done()
			logger.Debug("Log goroutine finished")
		}()
		for {
			select {
			case <-finish:
				return
			case rawMessage := <-receiver:
				log, err := createLokiLog(rawMessage, logger)
				if err == nil {
					sender <- log
					logger.Debug("Log message sent")
				} else {
					logger.Metadata(map[string]interface{}{
						"error": err,
					})
					logger.Error("Wrong log format received")
				}
			}
		}
	}()
}
