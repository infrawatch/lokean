package tests

import (
    "testing"
    "time"
    "reflect"
    "sync"

    "github.com/infrawatch/apputils/logging"
    "github.com/stretchr/testify/assert"

    "github.com/vyzigold/lokean/pkg/logs"
)

type Log struct {
    Labels map[string]string
    Message string
    Timestamp time.Duration
}

type TestSenderReciever struct {
    Message chan string
    Status chan int
    DoneChan chan bool
    LastLog Log
}

// Sender interface
func (tsr *TestSenderReciever) SendLog(labels map[string]string, message string, timestamp time.Duration) {
    tsr.LastLog = Log {
        Labels: labels,
        Message: message,
        Timestamp: timestamp,
    }
}

func (tsr *TestSenderReciever) Shutdown() {}

// Reciever interface
func (tsr *TestSenderReciever) GetNotifier() chan string {
    return tsr.Message
}

func (tsr *TestSenderReciever) GetStatus() chan int {
    return tsr.Status
}

func (tsr *TestSenderReciever) GetDoneChan() chan bool {
    return tsr.DoneChan
}

func (tsr *TestSenderReciever) Close() {}

// Other needed functions
func initialize() *TestSenderReciever {
    return &TestSenderReciever {
        Message: make(chan string),
        Status: make(chan int),
        DoneChan: make(chan bool),
        LastLog: Log{},
    }
}

func (tsr *TestSenderReciever) destroy() {
    close(tsr.Message)
    close(tsr.Status)
    close(tsr.DoneChan)
}

const TEST_LOG_FROM_AMQP = "{\"source\": \"abc\", \"level\": \"error\", \"timestamp\": 1588281700000, \"message\": \"hi\"}"

func TestLogs(t *testing.T) {
    tsr := initialize()

    logger, _ := logging.NewLogger(logging.DEBUG, "/dev/stderr")
    finish := make(chan bool)
    var wait sync.WaitGroup

    logs.Run(tsr, tsr, logger, finish, &wait)

    t.Run("Test send one log", func(t *testing.T) {
        // pretend that a log arrived
        tsr.Message <- TEST_LOG_FROM_AMQP
        tsr.Status <- 1

        // wait until the gorutine sends the log to loki
        time.Sleep(80 * time.Millisecond)

        // check, that the chanels are empty again
        assert.Equal(t, 0, len(tsr.Message))
        assert.Equal(t, 0, len(tsr.Status))

        // check, that the message got parsed as expected and
        // that it got sent to loki
        expectedLabels := make(map[string]string)
        expectedLabels["source"] = "abc"
        expectedLabels["level"] = "error"

        expectedLog := Log {
            Labels: expectedLabels,
            Timestamp: time.Duration(1588281700000) * time.Millisecond,
            Message: "hi",
        }

        assert.True(t, reflect.DeepEqual(expectedLog, tsr.LastLog))
    })
}
