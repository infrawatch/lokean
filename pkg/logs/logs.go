package logs

import (
	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/connector"

	"encoding/json"
	"sync"
	"time"
	"fmt"
	"strconv"
)

var severityArray = [...]string{
	"Emergency",
	"Alert",
	"Critical",
	"Error",
	"Warning",
	"Notice",
	"Informational",
	"Debug",
}

type Log struct {
	Msg             string        `json:"msg"`
	Rawmsg          string        `json:"rawmsg"`
	Timereported    time.Time        `json:"timereported"`
	Hostname        string        `json:"hostname"`
	Syslogtag       string        `json:"syslogtag"`
	Inputname       string        `json:"inputname"`
	Fromhost        string        `json:"fromhost"`
	FromhostIp      string        `json:"fromhost-ip"`
	Pri             string        `json:"pri"`
	Syslogfacility  string        `json:"syslogfacility"`
	Syslogseverity  string        `json:"syslogseverity"`
	Timegenerated   time.Time        `json:"timegenerated"`
	Programname     string        `json:"programname"`
	ProtocolVersion string        `json:"protocol-version"`
	StructuredData  interface{}   `json:"structured-data"`
	AppName         string        `json:"app-name"`
	Procid          string        `json:"procid"`
	Msgid           string        `json:"msgid"`
	Uuid            string        `json:"uuid"`
	Other           struct {
		Transport               string `json:"_TRANSPORT"`
		SystemdSlice            string `json:"_SYSTEMD_SLICE"`
		BootId                  string `json:"_BOOT_ID"`
		MachineId               string `json:"_MACHINE_ID"`
		Hostname                string `json:"_HOSTNAME"`
		Priority                string `json:"PRIORITY"`
		SyslogFacility          string `json:"SYSLOG_FACILITY"`
		SyslogIdentifier        string `json:"SYSLOG_IDENTIFIER"`
		Message                 string `json:"MESSAGE"`
		Uid                     string `json:"_UID"`
		Gid                     string `json:"_GID"`
		Comm                    string `json:"_COMM"`
		Exe                     string `json:"_EXE"`
		Cmdline                 string `json:"_CMDLINE"`
		CapEffective            string `json:"_CAP_EFFECTIVE"`
		SelinuxContext          string `json:"_SELINUX_CONTEXT"`
		SystedCgroup            string `json:"_SYSTED_CGROUP"`
		SystedUnit              string `json:"_SYSTED_UNIT"`
		SyslogTimestamp         string `json:"SYSLOG_TIMESTAMP"`
		Pid                     string `json:"_PID"`
		SystemdInvocationId     string `json:"_SYSTEMD_INVOCATION_ID"`
		SourceRealtimeTimestamp string `json:"_SOURCE_REALTIME_TIMESTAMP"`
	} `json:"$!"`
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
		labels["hostname"] = log.Hostname
		labels["programname"] = log.Programname

		severity, err := strconv.ParseInt(log.Syslogseverity, 10, 8)
		if err != nil {
			return connector.LokiLog{}, fmt.Errorf("Failed to parse severity: %s", err)
		}

		if severity > 7 || severity < 0 {
			return connector.LokiLog{}, fmt.Errorf("Unknown severity number: %d", severity)
		}

		logMessage := fmt.Sprintf("[%s] %s(%s) %s %s",
		                           severityArray[severity],
								   log.Other.Hostname,
								   log.FromhostIp,
								   log.Syslogtag,
								   log.Msg)
		return connector.LokiLog {
			Labels: labels,
			LogMessage: logMessage,
			Timestamp: time.Duration(log.Timereported.Unix()) * time.Second,
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
