package nginx

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type ServerBlock struct {
	ServerName []string
	Port       []string
}

type StatusMessage string

const (
	StatusMessageOK                        StatusMessage = "OK"
	StatusMessageNoSuchHost                StatusMessage = "NO_HOST"
	StatusMessageConnectionRefused         StatusMessage = "CONN_REFUSED"
	StatusMessageConnectionReset           StatusMessage = "CONN_RESET"
	StatusMessageTimeout                   StatusMessage = "TIMEOUT"
	StatusMessageUnknown                   StatusMessage = "UNKNOWN"
	StatusMessageFailedToVerifyCertificate StatusMessage = "FAILED_TO_VERIFY_CERTIFICATE"
)

type ServerStatus struct {
	ServerName    string        `json:"server_name" yaml:"server_name"`
	Port          int           `json:"port" yaml:"port"`
	StatusCode    int           `json:"status_code" yaml:"status_code"`
	StatusMessage StatusMessage `json:"status_message" yaml:"status_message"`
}

type ServerStatusDiff struct {
	ServerName     string        `json:"server_name" yaml:"server_name"`
	Port           int           `json:"port" yaml:"port"`
	OrigStatusCode int           `json:"orig_status_code" yaml:"orig_status_code"`
	NewStatusCode  int           `json:"new_status_code" yaml:"new_status_code"`
	StatusMessage  StatusMessage `json:"status_message" yaml:"status_message"`
}

type State struct {
	CreatedAt      time.Time      `json:"created_at" yaml:"created_at"`
	ConfigHash     string         `json:"config_hash" yaml:"config_hash"`
	ServerStatuses []ServerStatus `json:"server_statuses" yaml:"server_statuses"`
}

func protoForPort(port int) string {
	switch port {
	case 80:
		return "http"
	case 443:
		return "https"
	default:
		return ""
	}
}

func GetServerStatusWithTimeout(servername string, port int, timeout time.Duration) (ServerStatus, error) {
	l := log.WithFields(log.Fields{
		"app":        "nginx-ect",
		"func":       "GetServerStatus",
		"servername": servername,
		"port":       port,
	})
	l.Debug("getting server status")
	u := fmt.Sprintf("%s://%s:%d", protoForPort(port), servername, port)
	l.WithField("url", u).Debug("getting url")
	stat := ServerStatus{
		ServerName: servername,
		Port:       port,
	}
	c := http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		l.WithError(err).Debug("failed to create request")
		return stat, err
	}
	l.Debug("sending request")
	resp, err := c.Do(req)
	if err != nil {
		l.WithError(err).Debug("failed to send request")
		return stat, err
	}
	stat.StatusCode = resp.StatusCode
	stat.StatusMessage = StatusMessageOK
	return stat, nil
}
