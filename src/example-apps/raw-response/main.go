package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

const (
	CONN_HOST   = "0.0.0.0"
	CONN_TYPE   = "tcp"
	OUTPUT_FILE = "output-data"
)

func main() {
	// Listen for incoming connections.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	connString := fmt.Sprintf("%s:%s", CONN_HOST, port)
	l, err := net.Listen(CONN_TYPE, connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening: %s\n", err)
		os.Exit(1)
	}

	_, err = os.Open(OUTPUT_FILE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open 'output-data' to return data from: %s\n", err)
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Printf("Listening on %s\n", connString)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting: %s\n")
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go func() {
			err := handleRequest(conn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error handling request/response: %s\n", err)
			}
		}()
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) error {
	defer conn.Close()

	// Read the incoming connection into the buffer.
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	fmt.Printf("Received message:\n%s\n", buf)

	// re-open the output file every time, to make local testing easier
	outputFile, err := os.Open(OUTPUT_FILE)
	if err != nil {
		return err
	}
	// Send a response back to person contacting us.
	_, err = io.Copy(conn, outputFile)
	if err != nil {
		return err
	}
	fmt.Printf("Response sent\n")
	return nil
}
