package main

import (
    "github.com/vyzigold/lokean/internal/pkg/reciever"
    "github.com/vyzigold/lokean/internal/pkg/sender"
    "github.com/vyzigold/lokean/internal/pkg/logs"

    "github.com/infrawatch/apputils/connector"
    "github.com/infrawatch/apputils/logging"
    "github.com/infrawatch/apputils/config"
)

func main() {
    // TODO:
    // create a loki client using values from config
    // create an amqp client using values from config
    // logs.Run(amqpClient, lokiClient)
}
