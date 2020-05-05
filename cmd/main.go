package main

import (
	"time"
	"os"
	"fmt"
	"flag"
	"os/signal"
	"sync"

	"github.com/vyzigold/lokean/pkg/logs"

	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/config"
)

func printUsage() {
	fmt.Fprintln(os.Stderr, `Required command line argument missing`)
	flag.PrintDefaults()
}

//spawnSignalHandler spawns goroutine which will wait for interruption signal(s)
// and end lokean in case any of the signal is received
func spawnSignalHandler(finish chan bool, logger *logging.Logger, watchedSignals ...os.Signal) {
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, watchedSignals...)
	go func() {
	signalLoop:
		for sig := range interruptChannel {
			logger.Metadata(map[string]interface{}{
				"signal": sig,
			})
			logger.Error("Stopping execution on caught signal")
			close(finish)
			break signalLoop
		}
	}()
}

func parseLogLevel(s string) (logging.LogLevel, error) {
	if s == "DEBUG" {
		return logging.DEBUG, nil
	} else if s == "INFO" {
		return logging.INFO, nil
	} else if s == "WANGING" {
		return logging.WARN, nil
	} else if s == "ERROR" {
		return logging.ERROR, nil
	} else {
		return logging.ERROR, fmt.Errorf("Failed to parse the logLevel string: %s", s)
	}
}

func getConfigMetadata() map[string][]config.Parameter {
	elements := map[string][]config.Parameter{
		"default": []config.Parameter{
			config.Parameter{"logFile", "/dev/stderr", []config.Validator{}},
			config.Parameter{"logLevel", "INFO", []config.Validator{config.OptionsValidatorFactory([]string{"DEBUG", "INFO", "WARNING", "ERROR"})}},
		},
		"amqp1": []config.Parameter{
			config.Parameter{"url", "localhost:5672/lokean/logs", []config.Validator{}},
			config.Parameter{"messageCount", "1", []config.Validator{config.IntValidatorFactory()}},
			config.Parameter{"prefetch", "0", []config.Validator{config.IntValidatorFactory()}},
			config.Parameter{"name", "logs", []config.Validator{}},
		},
		"loki": []config.Parameter{
			config.Parameter{"url", "localhost:5672/lokean/logs", []config.Validator{}},
			config.Parameter{"batchSize", "2", []config.Validator{config.IntValidatorFactory()}},
			config.Parameter{"maxWaitTime", "100", []config.Validator{config.IntValidatorFactory()}},
		},
	}
	return elements
}

func main() {
	flag.Usage = printUsage
	fConfigLocation := flag.String("config", "", "Path to configuration file.")
	flag.Parse()

	if len(*fConfigLocation) == 0 {
		printUsage()
		os.Exit(1)
	}

	// init logger with temporary values until the correct ones
	// can be read from config
	logger, err := logging.NewLogger(logging.ERROR, "/dev/stderr")
	if err != nil {
		fmt.Printf("Failed to open tempLogger: %s\n", err.Error())
		os.Exit(1)
	}
	defer logger.Destroy()

	metadata := getConfigMetadata()
	conf, err := config.NewConfig(metadata, logger)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
		})
		logger.Error("Failed to initialize config reader")
		os.Exit(1)
	}

	err = conf.Parse(*fConfigLocation)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
			"file": *fConfigLocation,
		})
		logger.Error("Failed to parse the config file")
		os.Exit(1)
	}

	logFile := conf.Sections["default"].Options["logFile"].GetString()
	logLevelString := conf.Sections["default"].Options["logLevel"].GetString()
	logLevel, err := parseLogLevel(logLevelString)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
			"logLevel": logLevelString,
		})
		logger.Error("Failed to parse log level from config file")
		os.Exit(1)
	}
	logger.SetLogLevel(logLevel)
	err = logger.SetFile(logFile, 0666)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
			"logFile": logFile,
		})
		logger.Error("Failed to set proper log ifle")
		os.Exit(1)
	}

	amqpURL := conf.Sections["amqp1"].Options["url"].GetString()
	amqpMsgcount := conf.Sections["amqp1"].Options["messageCount"].GetInt()
	amqpPrefetch := conf.Sections["amqp1"].Options["prefetch"].GetInt()
	amqpName := conf.Sections["amqp1"].Options["name"].GetString()

	lokiURL := conf.Sections["loki"].Options["url"].GetString()
	lokiBatchSize := conf.Sections["loki"].Options["batchSize"].GetInt()
	lokiMaxWait := conf.Sections["loki"].Options["maxWaitTime"].GetInt()

	finish := make(chan bool)
	var wait sync.WaitGroup
	spawnSignalHandler(finish, logger, os.Interrupt)

	amqpClient := connector.NewAMQPServer(
		amqpURL,
		amqpMsgcount,
		amqpPrefetch,
		amqpName,
		logger)

	lokiClient, err := connector.NewLokiConnector(
		lokiURL,
		lokiBatchSize,
		time.Duration(lokiMaxWait) * time.Millisecond)

	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
			"url": lokiURL,
		})
		logger.Error("Couldn't create a loki client.")
		os.Exit(1)
	}

	lokiClient.Start()
	defer lokiClient.Shutdown()

	logs.Run(amqpClient, lokiClient, logger, finish, &wait)

	wait.Wait()
}
