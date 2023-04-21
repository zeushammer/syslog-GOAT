package main

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestSyslogServer(t *testing.T) {
	// Create a new syslog server
	server := NewSyslogServer(":7531")
	go server.Start()

	time.Sleep(time.Second * 3)
	// Connect to the server using UDP
	conn, err := net.Dial("udp", ":7531")
	if err != nil {
		t.Fatalf("Error connecting to UDP server: %s", err)
	}
	defer conn.Close()

	// Send a syslog message to the server
	message := []byte("<13>Apr 20 15:04:05 hostname myapp: message")
	_, err = conn.Write(message)
	if err != nil {
		t.Fatalf("Error sending message to UDP server: %s", err)
	}

	// Wait for the message to be received by the server
	select {
	case received := <-server.Messages:
		// Parse the received message into a map
		var parsed map[string]interface{}
		err = json.Unmarshal(received, &parsed)
		if err != nil {
			t.Fatalf("Error parsing received message: %s", err)
		}

		// Verify that the parsed message matches the original message
		expected := map[string]interface{}{
			"severity": 5,
			"facility": 1,
			"hostname": "hostname",
			"message":  "myapp: message",
		}
		if !reflect.DeepEqual(parsed, expected) {
			t.Errorf("Received message %v does not match expected %v", parsed, expected)
		}
	case <-time.After(time.Second):
		t.Errorf("Timed out waiting for message to be received by server")
	}

	// Stop the server
	server.Stop()
}
