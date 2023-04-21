package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type SyslogMessage struct {
	Facility int    `json:"facility"`
	Severity int    `json:"severity"`
	Hostname string `json:"hostname"`
	Message  string `json:"message"`
}

type Severity int

const (
	Emergency Severity = iota
	Alert
	Critical
	Error
	Warning
	Notice
	Informational
	Debug
)

type Facility int

const (
	KernelMessages Facility = iota << 3
	UserLevelMessages
	MailSystem
	SystemDaemons
	SecurityOrAuthorizationMessages
	MessagesGeneratedInternallyBySyslogd
	LinePrinterSubsystem
	NetworkNewsSubsystem
	UUCPSubsystem
	ClockDaemon
	SecurityOrAuthorizationMessages2
	FTPDaemon
	NTPSubsystem
	LogAudit
	LogAlert
	ClockDaemon2
	LocalUse0
	LocalUse1
	LocalUse2
	LocalUse3
	LocalUse4
	LocalUse5
	LocalUse6
	LocalUse7
)

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

func main() {
	addr := ":7531"
	server := NewSyslogServer(addr)
	go server.Run()

	// Block the main goroutine so that the server can continue running
	select {}
}

type SyslogServer struct {
	Conn   *net.UDPConn
	Addr     string
	Messages chan []byte
}

// func (s *SyslogServer) Run() (err error) { return nil }
// func (s *SyslogServer) Close() error     { return nil }

func NewSyslogServer(addr string) *SyslogServer {
	return &SyslogServer{
		Addr:     addr,
		Messages: make(chan []byte),
	}
}

func (s *SyslogServer) Close() error {
	return s.Conn.Close()
}

func (s *SyslogServer) Run() {
	udpAddr, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		logger.Fatalf("Error resolving UDP address: %s", err)
	}

	s.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.Fatalf("Error listening on UDP address: %s", err)
	}

	logger.Printf("Syslog server listening on %s", s.Addr)

	err = s.handleMessages()
	if err != nil {
		logger.Fatalf("Error handling connections: %s", err)
	}
}

func (s *SyslogServer) handleMessages() error {
	var err error
	for {
		scanner := bufio.NewScanner(s.Conn)
		for scanner.Scan() {
			message := scanner.Text()
			log.Println("Received message:", message)
			go s.handleMessage(message)
		}

		if err := scanner.Err(); err != nil {
			logger.Fatalf("Scanner error: %s", err)
		}
	}
	return err
}

func (s *SyslogServer) handleMessage(message string) {
	parsedMessage, err := parseSyslogMessage(message)
	if err != nil {
		log.Println("Error parsing syslog message:", err)
	}
	log.Println("Parsed message:", string(parsedMessage))
	s.Messages <- parsedMessage // need to use select for best effort
}

func parseSyslogMessage(msg string) ([]byte, error) {
	const severityMask = 0x07
	const facilityShift = 3
	const facilityMask = 0xf8

	// Parse the priority value from the message

	if !strings.HasPrefix(msg, "<") || !strings.Contains(msg, ">") {
		logger.Fatalf("invalid priority value")
	}

	priorityValue := (msg)[1:strings.Index(msg, ">")]
	priorityNum, err := strconv.Atoi(priorityValue)
	if err != nil {
		logger.Fatalf("strconv.Atoi error: %s", err)
	}

	// Parse the severity and facility values from the priority

	if priorityNum < 0 || priorityNum > 191 {
		logger.Fatalf("invalid priority value: %d", priorityNum)
	}

	severityVal := Severity(priorityNum & severityMask)
	if severityVal < Emergency || severityVal > Debug {
		logger.Fatalf("invalid severity value: %d", severityVal)
	}

	facilityVal := Facility((priorityNum & facilityMask) >> facilityShift)

	// Extract the hostname and message from the message string
	hostname := "unknown"
	message := ""
	if i := strings.Index(msg, " "); i != -1 {
		hostname = msg[4:i]
		message = msg[i+1:]
	}

	// Create a map with the parsed values
	parsed := map[string]interface{}{
		"severity": severityVal,
		"facility": facilityVal,
		"hostname": hostname,
		"message":  message,
	}

	// Convert the map to JSON
	jsonMessage, err := json.Marshal(parsed)
	if err != nil {
		return nil, err
	}

	return jsonMessage, nil
}
