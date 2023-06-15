package nginx

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robertlestak/nginx-ect/internal/utils"
	log "github.com/sirupsen/logrus"
)

type NginxEct struct {
	ConfigFilePath       string
	ConfigStr            string
	ConfigHash           string
	StateFilePath        string
	LoadedConfigHash     string
	Timeout              time.Duration
	ServerBlocks         []ServerBlock
	ServerStatuses       []ServerStatus
	LoadedServerStatuses []ServerStatus
	Diffs                []ServerStatusDiff
	Concurrency          int
	VerifyHash           bool
	IgnoreServers        []string
}

func (n *NginxEct) LoadConfigFile() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "LoadConfigFile",
		"file": n.ConfigFilePath,
	})
	l.Debug("loading config file")
	fd, err := utils.ReadFile(n.ConfigFilePath)
	if err != nil {
		l.WithError(err).Error("failed to read config file")
		return err
	}
	n.ConfigStr = fd
	l.Debug("loaded config file")
	return nil
}

func hashConfig(cfg string) (string, error) {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "hashConfig",
	})
	l.Debug("hashing config")
	h := sha256.New()
	if _, err := h.Write([]byte(cfg)); err != nil {
		l.WithError(err).Error("failed to hash config")
		return "", err
	}
	l.WithField("hash", fmt.Sprintf("%x", h.Sum(nil))).Debug("hashed config")
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (n *NginxEct) HashConfig() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "HashConfig",
	})
	l.Debug("hashing config")
	h, err := hashConfig(n.ConfigStr)
	if err != nil {
		l.WithError(err).Error("failed to hash config")
		return err
	}
	l.WithField("hash", h).Debug("hashed config")
	n.ConfigHash = h
	return nil
}

func (n *NginxEct) LoadStateFile() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "LoadStateFile",
		"file": n.StateFilePath,
	})
	l.Debug("loading state file")
	state := &State{}
	fd, err := utils.ReadFile(n.StateFilePath)
	if err != nil {
		l.WithError(err).Error("failed to read state file")
		return err
	}
	if err := json.Unmarshal([]byte(fd), &state); err != nil {
		l.WithError(err).Error("failed to unmarshal state file")
		return err
	}
	n.LoadedServerStatuses = state.ServerStatuses
	n.LoadedConfigHash = state.ConfigHash
	l.Debug("loaded state file")
	return nil
}

func (n *NginxEct) ParseConfig() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "ParseConfig",
	})
	l.Debug("parsing config")
	// p := parser.NewStringParser(n.ConfigStr)
	// n.Config = p.Parse()
	// Regular expressions to match server_name and listen directives
	serverNameRegex := regexp.MustCompile(`server_name\s+([^;]+);`)
	listenRegex := regexp.MustCompile(`listen\s+([^;]+);`)

	// Slice to store ServerBlock structs
	serverBlocks := []ServerBlock{}

	// Read the file line by line
	scanner := bufio.NewScanner(strings.NewReader(n.ConfigStr))
	var currentBlock ServerBlock
	for scanner.Scan() {
		line := scanner.Text()

		// Check for server block start
		if strings.Contains(line, "server {") {
			currentBlock = ServerBlock{}
		}
		if matches := serverNameRegex.FindStringSubmatch(line); len(matches) > 0 {
			servers := strings.Split(matches[1], " ")
			for _, server := range servers {
				ignored := utils.Contains(n.IgnoreServers, server)
				if server != "" && server != "on" && server != "_" && server != "localhost" && !ignored {
					currentBlock.ServerName = append(currentBlock.ServerName, server)
				}
			}
		}

		// Check for listen directive
		if matches := listenRegex.FindStringSubmatch(line); len(matches) > 0 {
			ports := strings.Split(matches[1], " ")
			for _, port := range ports {
				if _, err := strconv.Atoi(port); err == nil {
					currentBlock.Port = append(currentBlock.Port, port)
				}
			}
		}

		// Check for server block end
		if strings.Contains(line, "}") && len(currentBlock.ServerName) > 0 && len(currentBlock.Port) > 0 {
			serverBlocks = append(serverBlocks, currentBlock)
		}
	}
	if err := scanner.Err(); err != nil {
		l.WithError(err).Error("failed to scan config file")
		return err
	}
	var uniqueServerBlocks []ServerBlock
	// ensure each server block is unique
	for _, block := range serverBlocks {
		unique := true
		for _, uniqueBlock := range uniqueServerBlocks {
			if reflect.DeepEqual(block, uniqueBlock) {
				unique = false
				break
			}
		}
		if unique {
			uniqueServerBlocks = append(uniqueServerBlocks, block)
		}
	}
	n.ServerBlocks = uniqueServerBlocks
	l.Debug("parsed config")
	return nil
}

func (n *NginxEct) Init() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "Init",
	})
	l.Debug("initializing")
	if err := n.LoadConfigFile(); err != nil {
		l.WithError(err).Error("failed to load config file")
		return err
	}
	if err := n.ParseConfig(); err != nil {
		l.WithError(err).Error("failed to parse config")
		return err
	}
	if err := n.FilterSupportedProtocols(); err != nil {
		l.WithError(err).Error("failed to filter supported protocols")
		return err
	}
	return nil
}

func (n *NginxEct) FilterSupportedProtocols() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "FilterSupportedProtocols",
	})
	supportedPorts := []string{"80", "443"}
	l.Debug("filtering supported protocols")
	cleanBlocks := []ServerBlock{}
	for _, serverBlock := range n.ServerBlocks {
		l.Debug("checking server block")
		for _, port := range serverBlock.Port {
			l.Debug("checking port")
			if utils.Contains(supportedPorts, port) {
				l.Debug("found supported port")
				cleanBlocks = append(cleanBlocks, serverBlock)
				break
			}
		}
	}
	n.ServerBlocks = cleanBlocks
	return nil
}

func statusWorker(timeout time.Duration, jobs chan ServerBlock, results chan ServerStatus) {
	for sb := range jobs {
		l := log.WithFields(log.Fields{
			"app":  "nginx-ect",
			"func": "statusWorker",
		})
		l.Debug("checking server status")
		for _, serverName := range sb.ServerName {
			for _, port := range sb.Port {
				l.WithFields(log.Fields{
					"serverName": serverName,
					"port":       port,
				}).Debug("checking server status")
				intPort, err := strconv.Atoi(port)
				if err != nil {
					l.WithError(err).Error("failed to convert port to int")
					results <- ServerStatus{
						ServerName:    serverName,
						Port:          0,
						StatusCode:    0,
						StatusMessage: StatusMessage(fmt.Sprintf("failed to convert port to int: %s", err.Error())),
					}
					continue
				}
				stat, err := GetServerStatusWithTimeout(serverName, intPort, timeout)
				if err != nil {
					l.WithError(err).Debug("server status check failed")
					stat := ServerStatus{
						ServerName: serverName,
						Port:       intPort,
						StatusCode: 0,
					}
					if strings.Contains(err.Error(), "timeout") {
						stat.StatusMessage = StatusMessageTimeout
					} else if strings.Contains(err.Error(), "connection refused") {
						stat.StatusMessage = StatusMessageConnectionRefused
					} else if strings.Contains(err.Error(), "no such host") {
						stat.StatusMessage = StatusMessageNoSuchHost
					} else if strings.Contains(err.Error(), "failed to verify certificate: x509") {
						stat.StatusMessage = StatusMessageFailedToVerifyCertificate
					} else {
						stat.StatusMessage = StatusMessageUnknown
					}
					results <- stat
					continue
				}
				results <- stat
			}
		}
	}
}

func (n *NginxEct) totalJobs() int {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "totalJobs",
	})
	l.Debug("calculating total jobs")
	total := 0
	for _, sb := range n.ServerBlocks {
		total += len(sb.ServerName) * len(sb.Port)
	}
	l.WithField("total", total).Debug("total jobs")
	return total
}

func (n *NginxEct) IndexNginx() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "IndexNginx",
	})
	l.Debug("indexing")
	l.Debug("indexing config")
	jobs := make(chan ServerBlock, n.totalJobs())
	results := make(chan ServerStatus, n.totalJobs())
	for w := 1; w <= n.Concurrency; w++ {
		go statusWorker(n.Timeout, jobs, results)
	}
	for _, sb := range n.ServerBlocks {
		jobs <- sb
	}
	close(jobs)
	for a := 1; a <= n.totalJobs(); a++ {
		result := <-results
		l.WithFields(log.Fields{
			"serverName": result.ServerName,
			"port":       result.Port,
			"status":     result.StatusCode,
			"total":      n.totalJobs(),
			"current":    a,
		}).Debug("got server status")
		n.ServerStatuses = append(n.ServerStatuses, result)
	}
	return nil
}

func (n *NginxEct) Index() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "Index",
	})
	l.Debug("indexing")
	if err := n.Init(); err != nil {
		l.WithError(err).Error("failed to initialize")
		return err
	}
	if err := n.HashConfig(); err != nil {
		l.WithError(err).Error("failed to hash config")
		return err
	}
	if err := n.IndexNginx(); err != nil {
		l.WithError(err).Error("failed to index nginx")
		return err
	}
	if err := n.GenerateStateFile(); err != nil {
		l.WithError(err).Error("failed to generate state file")
		return err
	}
	return nil
}

func (n *NginxEct) GenerateStateFile() error {
	l := log.WithFields(log.Fields{
		"app":       "nginx-ect",
		"func":      "GenerateStateFile",
		"statefile": n.StateFilePath,
	})
	l.Debug("generating state file")
	state := &State{
		CreatedAt:      time.Now().UTC(),
		ServerStatuses: n.ServerStatuses,
		ConfigHash:     n.ConfigHash,
	}
	jd, err := json.Marshal(state)
	if err != nil {
		l.WithError(err).Error("failed to marshal json")
		return err
	}
	if err := utils.WriteFile(n.StateFilePath, string(jd)); err != nil {
		l.WithError(err).Error("failed to write state file")
		return err
	}
	return nil
}

func (n *NginxEct) DiffState() error {
	l := log.WithFields(log.Fields{
		"app":       "nginx-ect",
		"func":      "DiffState",
		"statefile": n.StateFilePath,
		"statuses":  len(n.ServerStatuses),
		"loaded":    len(n.LoadedServerStatuses),
	})
	l.Debug("diffing state")
	// compare ServerStatuses with LoadedServerStatuses
	for _, ss := range n.ServerStatuses {
		for _, lss := range n.LoadedServerStatuses {
			if ss.ServerName == lss.ServerName && ss.Port == lss.Port {
				l.WithFields(log.Fields{
					"serverName": ss.ServerName,
					"port":       ss.Port,
					"origStatus": lss.StatusCode,
					"newStatus":  ss.StatusCode,
					"message":    ss.StatusMessage,
				}).Debug("found server status")
				if ss.StatusCode != lss.StatusCode {
					sd := ServerStatusDiff{
						ServerName:     ss.ServerName,
						Port:           ss.Port,
						OrigStatusCode: lss.StatusCode,
						NewStatusCode:  ss.StatusCode,
						StatusMessage:  ss.StatusMessage,
					}
					n.Diffs = append(n.Diffs, sd)
				}
			}
		}
	}
	if len(n.Diffs) > 0 {
		l.WithField("diffs", len(n.Diffs)).Debug("found diffs")
	} else {
		l.Debug("no diffs found")
	}
	return nil
}

func (n *NginxEct) Diff() error {
	l := log.WithFields(log.Fields{
		"app":  "nginx-ect",
		"func": "Diff",
	})
	l.Debug("diffing")
	if err := n.Init(); err != nil {
		l.WithError(err).Error("failed to initialize")
		return err
	}
	if err := n.LoadStateFile(); err != nil {
		l.WithError(err).Error("failed to load state file")
		return err
	}
	if err := n.HashConfig(); err != nil {
		l.WithError(err).Error("failed to hash config")
		return err
	}
	if n.VerifyHash {
		// ensuring hashs match
		if n.ConfigHash != n.LoadedConfigHash {
			l.Errorf("config hash mismatch: %s != %s", n.ConfigHash, n.LoadedConfigHash)
			return errors.New("config hash mismatch")
		} else {
			l.Debug("config hash matches")
		}
	} else {
		l.Warn("skipping config hash verification")
	}
	l.Debug("diffing config")
	if err := n.IndexNginx(); err != nil {
		l.WithError(err).Error("failed to index")
		return err
	}
	l.Debug("diffing state")
	if err := n.DiffState(); err != nil {
		l.WithError(err).Error("failed to diff state")
		return err
	}
	if len(n.Diffs) > 0 {
		l.Errorf("found %d diffs", len(n.Diffs))
		for _, d := range n.Diffs {
			l.WithFields(log.Fields{
				"serverName":     d.ServerName,
				"port":           d.Port,
				"origStatusCode": d.OrigStatusCode,
				"newStatusCode":  d.NewStatusCode,
				"statusMessage":  d.StatusMessage,
			}).Error("server status changed")
		}
		return errors.New("found diffs")
	}
	l.Debug("no diffs found")
	return nil
}
