package logs

import (
    "github.com/vyzigold/lokean/internal/reciever"
    "github.com/vyzigold/lokean/internal/sender"
)

func Run(reciever Reciever, sender Sender) {
    // TODO:
    // create a loop which waits for reciever, then parses
    // each message to create labels for each logs
    // and then give each log to sender
}
