package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/infrawatch/apputils/config"
	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/system"

	"github.com/infrawatch/lokean/pkg/sources"
)

func printUsage() {
	fmt.Fprintln(os.Stderr, `Required command line argument missing`)
	flag.PrintDefaults()
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
			config.Parameter{Name: "logFile", Tag: "", Default: "/dev/stderr", Validators: []config.Validator{}},
			config.Parameter{Name: "logLevel", Tag: "", Default: "INFO", Validators: []config.Validator{config.StringOptionsValidatorFactory([]string{"DEBUG", "INFO", "WARNING", "ERROR"})}},
			config.Parameter{Name: "message_transport", Tag: "", Default: "amqp1", Validators: []config.Validator{config.StringOptionsValidatorFactory([]string{"amqp1", "socket"})}},
		},
		"amqp1": []config.Parameter{
			config.Parameter{Name: "connection", Tag: "", Default: "amqp://localhost:5672", Validators: []config.Validator{}},
			config.Parameter{Name: "send_timeout", Tag: "", Default: 2, Validators: []config.Validator{config.IntValidatorFactory()}},
			config.Parameter{Name: "client_name", Tag: "", Default: "test", Validators: []config.Validator{}},
			config.Parameter{Name: "listen_channels", Tag: "", Default: "lokean/logs:rsyslog", Validators: []config.Validator{}},
			config.Parameter{Name: "listen_prefetch", Tag: "", Default: -1, Validators: []config.Validator{config.IntValidatorFactory()}},
		},
		"socket": []config.Parameter{
			config.Parameter{Name: "in_address", Tag: "", Default: "/tmp/lokean", Validators: []config.Validator{}},
		},
		"loki": []config.Parameter{
			config.Parameter{Name: "connection", Tag: "", Default: "http://localhost:3100", Validators: []config.Validator{}},
			config.Parameter{Name: "batch_size", Tag: "", Default: 20, Validators: []config.Validator{config.IntValidatorFactory()}},
			config.Parameter{Name: "max_wait_time", Tag: "", Default: 100, Validators: []config.Validator{config.IntValidatorFactory()}},
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
	conf := config.NewINIConfig(metadata, logger)

	err = conf.Parse(*fConfigLocation)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
			"file":  *fConfigLocation,
		})
		logger.Error("Failed to parse the config file")
		os.Exit(1)
	}

	logFile := conf.Sections["default"].Options["logFile"].GetString()
	logLevelString := conf.Sections["default"].Options["logLevel"].GetString()
	logLevel, err := parseLogLevel(logLevelString)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error":    err,
			"logLevel": logLevelString,
		})
		logger.Error("Failed to parse log level from config file")
		os.Exit(1)
	}
	logger.SetLogLevel(logLevel)
	err = logger.SetFile(logFile, 0666)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error":   err,
			"logFile": logFile,
		})
		logger.Error("Failed to set proper log file")
		os.Exit(1)
	}

	finish := make(chan bool)
	var wait sync.WaitGroup
	system.SpawnSignalHandler(finish, logger, os.Interrupt)

	transportOpt, err := conf.GetOption("default/message_transport")
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
		})
		logger.Error("Failed to get message transport option")
		os.Exit(1)
	}
	transportReceiver := make(chan interface{})
	transportSender := make(chan interface{})
	transport := transportOpt.GetString()
	switch transport {
	case "socket":
		socket, err := connector.ConnectUnixSocket(conf, logger)
		if err != nil {
			logger.Metadata(map[string]interface{}{"error": err})
			logger.Error("Couldn't connect to socket")
			os.Exit(1)
		}
		socket.Start(transportReceiver, transportSender)
		defer socket.Disconnect()
	case "amqp1":
		amqp, err := connector.ConnectAMQP10(conf, logger)
		if err != nil {
			logger.Metadata(map[string]interface{}{"error": err})
			logger.Error("Couldn't connect to AMQP")
			os.Exit(1)
		}
		amqp.Start(transportReceiver, transportSender)
		defer amqp.Disconnect()
	}

	loki, err := connector.ConnectLoki(conf, logger)
	if err != nil {
		logger.Metadata(map[string]interface{}{
			"error": err,
		})
		logger.Error("Couldn't connect to Loki")
		return
	}
	lokiReceiver := make(chan interface{})
	lokiSender := make(chan interface{})
	loki.Start(lokiReceiver, lokiSender)
	defer loki.Disconnect()

	sources.Run(transportReceiver, lokiSender, logger, finish, &wait)

	wait.Wait()
}
