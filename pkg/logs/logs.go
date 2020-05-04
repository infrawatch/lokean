package logs

import (
    "github.com/vyzigold/lokean/pkg/reciever"
    "github.com/vyzigold/lokean/pkg/sender"
    "github.com/infrawatch/apputils/logging"

    "encoding/json"
    "sync"
    "time"
)

type Log struct {
    Level string `json:"level"`
    Timestamp int `json:"timestamp"`
    Source string `json:"source"`
    LogMessage string `json:"message"`
}

func Run(reciever reciever.Reciever, sender sender.Sender, logger *logging.Logger, finish chan bool, wait *sync.WaitGroup) {
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
            case recieverStatus := <-reciever.GetStatus():
                if recieverStatus != 1 {
                    logger.Error("Recieved a bad reciever status, shutting down")
                    break
                }
            case rawMessage := <-reciever.GetNotifier():
                logger.Debug("Recieved the folowing log message:")
                logger.Debug(rawMessage)
                var log Log
                err := json.Unmarshal([]byte(rawMessage), &log)
                if err != nil {
                    logger.Error("Wrong log format recieved")
                    continue
                }
                labels := make(map[string]string)
                labels["level"] = log.Level
                labels["source"] = log.Source
                sender.SendLog(labels, log.LogMessage, time.Duration(log.Timestamp) * time.Millisecond)
                logger.Debug("Log message successfuly sent")

            }
        }
    }()
}
