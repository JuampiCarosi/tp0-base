package common

import (
	"fmt"
	"log"
	"net"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/shared"
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
		log.Printf("action: connection_closed | result: success | connection: %v", s.clientConn.LocalAddr())
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
	errorResponse := shared.BetResponse(false)
	errorResponseSerialized, err := errorResponse.Serialize()

	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		return
	}

	messageType, err := shared.MessageFromSocket(&s.clientConn)
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		s.clientConn.Write(errorResponseSerialized)
		return
	}

	if messageType.Type != shared.BetType {
		log.Printf("action: handle_client_connection | result: fail | error: unknown message type %v", messageType.Type)
		s.clientConn.Write(errorResponseSerialized)
		return
	}

	var betMessage shared.BetMessage
	err = betMessage.Deserialize(messageType.Payload)
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		s.clientConn.Write(errorResponseSerialized)
		return
	}
	bet := betMessage.ReceivedBet
	err = bets.StoreBets([]bets.Bet{bet})

	if err != nil {
		log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
		s.clientConn.Write(errorResponseSerialized)
		return
	}

	successResponse := shared.BetResponse(true)
	successResponseSerialized, err := successResponse.Serialize()
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		s.clientConn.Write(errorResponseSerialized)
		return
	}

	log.Printf("action: apuesta_almacenada | result: success | dni: %v | numero: %v", bet.Document, bet.Number)
	s.clientConn.Write(successResponseSerialized)
}
