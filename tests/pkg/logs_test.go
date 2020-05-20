package tests

import (
	"testing"
	"time"
	"reflect"
	"sync"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/connector"
	"github.com/stretchr/testify/assert"

	"github.com/vyzigold/lokean/pkg/logs"
)


const TEST_LOG_FROM_AMQP = "{\"source\": \"abc\", \"level\": \"error\", \"timestamp\": 1588281700000, \"message\": \"hi\"}"

func TestLogs(t *testing.T) {
	mockedAMQPReceiver := make(chan interface{})
	mockedLokiSender := make(chan interface{})

	logger, _ := logging.NewLogger(logging.DEBUG, "/dev/stderr")
	finish := make(chan bool)
	var wait sync.WaitGroup

	logs.Run(mockedAMQPReceiver, mockedLokiSender, logger, finish, &wait)

	t.Run("Test send one log", func(t *testing.T) {
		// pretend that a log arrived
		mockedAMQPReceiver <- connector.AMQP10Message {
			Body: TEST_LOG_FROM_AMQP,
		}

		// wait until the gorutine sends the log to loki
		time.Sleep(80 * time.Millisecond)

		// check, that the channel is empty again
		assert.Equal(t, 0, len(mockedAMQPReceiver))

		// check, that the message got parsed as expected and
		// that it got sent to loki
		expectedLabels := make(map[string]string)
		expectedLabels["source"] = "abc"

		expectedLog := connector.LokiLog {
			Labels: expectedLabels,
			Timestamp: time.Duration(1588281700000) * time.Millisecond,
			LogMessage: "[error] hi",
		}

		assert.True(t, reflect.DeepEqual(expectedLog, <-mockedLokiSender))
	})
}
