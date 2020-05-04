package sender

import (
    "time"
)

type Sender interface {
    // I'm not sure how are logs stored in other applications
    // (other than loki), so I left a map[string]string there
    // for assigning key=value labels like in loki
    // TODO add this to the loki client in apputils
    SendLog(map[string]string, string, time.Duration)
    Shutdown()
}
