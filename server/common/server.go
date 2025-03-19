package common

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	serverSocket net.Listener
	running      bool
	clientConn   net.Conn
}

func NewServer(address string) (*Server, error) {
	server := &Server{
		running: true,
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("error creating server socket: %v", err)
	}
	server.serverSocket = listener

	return server, nil
}

func (s *Server) Run() {
	for s.running {
		clientConn, err := s.acceptNewConnection()
		if err != nil {
			log.Printf("action: accept_connections | result: failed | error: %v", err)
			continue
		}
		s.clientConn = clientConn
		s.handleClientConnection()
		s.clientConn = nil
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.running = false
	if s.clientConn != nil {
		s.clientConn.Close()
		log.Printf("action: connection closed | result: success | connection: %v", s.clientConn.LocalAddr())
	}
	if s.serverSocket != nil {
		s.serverSocket.Close()
		log.Printf("action: server_socket_closed | result: success")
	}
}

func (s *Server) acceptNewConnection() (net.Conn, error) {
	log.Printf("action: accept_connections | result: in_progress")
	conn, err := s.serverSocket.Accept()
	if err != nil {
		return nil, err
	}
	log.Printf("action: accept_connections | result: success | ip: %v", conn.RemoteAddr().String())
	return conn, nil
}

func (s *Server) handleClientConnection() {
	defer s.clientConn.Close()

	// buffer := make([]byte, 1024)
	// n, err := s.clientConn.Read(buffer)
	// if err != nil {
	// 	log.Printf("action: receive_message | result: fail | error: %v", err)
	// 	return
	// }

	// log.Printf("action: receive_message | result: success | n: '%v', buffer: '%v'", n, string(buffer))

	buffer := make([]byte, 1024)
	var message []byte
	read, err := s.clientConn.Read(buffer)
	message = append(message, buffer...)
	if err != nil {
		log.Printf("action: receive_message | result: fail | error: %v", err)
		return
	}

	for strings.Count(string(message), " ") < 1 {
		curr, err2 := s.clientConn.Read(buffer)
		read += curr
		message = append(message, buffer...)

		if len(buffer) == 0 {
			log.Printf("action: receive_message | result: fail | error: %v", err2)
			return
		}

		if err2 != nil {
			log.Printf("action: receive_message | result: fail | error: %v", err2)
			return
		}
	}
	split := strings.SplitN(string(message), " ", 2)
	lengthStr := split[0]
	length, err := strconv.Atoi(lengthStr)
	payload := make([]byte, length)
	copy(payload, split[1])
	if err != nil {
		log.Printf("action: receive_message | result: fail | error: %v", err)
		return
	}
	for read < length {
		curr, err2 := s.clientConn.Read(payload[read:])
		read += curr
		if err2 != nil {
			log.Printf("action: receive_message | result: fail | error: %v", err2)
			return
		}
	}

	bet, err := parseBet(string(payload))
	if err != nil {
		log.Printf("action: parse_bet | result: fail | error: %v", err)
		s.sendResponse("ERROR SAVING BET\n")
		return
	}

	err = StoreBets([]*Bet{bet})
	if err != nil {
		log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
		s.sendResponse("ERROR SAVING BET\n")
		return
	}

	log.Printf("action: apuesta_almacenada | result: success | dni: %v | numero: %v", bet.Document, bet.Number)
	s.sendResponse("OK\n")
}

func (s *Server) sendResponse(response string) {
	payload := []byte(response)
	written := 0
	for written < len(payload) {
		n, err := s.clientConn.Write(payload[written:])
		if err != nil {
			log.Printf("Error sending response: %v", err)
			return
		}
		written += n
	}
}

func parseBet(msg string) (*Bet, error) {
	parts := strings.Split(msg, ";")
	log.Printf("action: parse_bet | result: success | msg: %v | parts: %v", msg, parts)
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid message format")
	}

	agency, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid agency: %v", err)
	}

	birthDate, err := time.Parse("2006-01-02", parts[4])
	if err != nil {
		return nil, fmt.Errorf("invalid birthdate: %v", err)
	}

	number, err := strconv.Atoi(parts[5])
	if err != nil {
		return nil, fmt.Errorf("invalid number: %v", err)
	}

	return &Bet{
		Agency:    agency,
		FirstName: parts[1],
		LastName:  parts[2],
		Document:  parts[3],
		BirthDate: birthDate,
		Number:    number,
	}, nil
}
