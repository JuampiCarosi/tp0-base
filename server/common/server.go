package common

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/shared"
)

type Server struct {
	serverSocket     net.Listener
	running          bool
	clientConn       net.Conn
	totalAgencies    int
	receivedAgencies int
	winners          map[int][]string
}

func NewServer(address string, agenciesAmount int) (*Server, error) {
	server := &Server{
		running:          true,
		totalAgencies:    agenciesAmount,
		receivedAgencies: 0,
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
			return
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

	switch messageType.Type {
	case shared.BetType:
		s.handleBetMessage(messageType)
	case shared.BatchBetType:
		s.handleBatchBetMessage(messageType)
	case shared.AllBetsSentType:
		s.handleAllBetsSentMessage()
	case shared.ResultsQueryType:
		s.handleResultsQueryMessage(messageType)
	default:
		log.Printf("action: handle_client_connection | result: fail | error: unknown message type %v", messageType.Type)
		s.clientConn.Write(errorResponseSerialized)
		return
	}

}

func (s *Server) handleBetMessage(message *shared.RawMessage) {
	errorResponse := shared.BetResponse(false)
	errorResponseSerialized, err := errorResponse.Serialize()
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		s.clientConn.Write(errorResponseSerialized)
		return
	}

	var betMessage shared.BetMessage
	err = betMessage.Deserialize(message.Payload)
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		s.clientConn.Write(errorResponseSerialized)
		return
	}
	bet := betMessage.ReceivedBet
	err = bets.StoreBets([]*bets.Bet{&bet})

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

func (s *Server) handleBatchBetMessage(message *shared.RawMessage) {

	var batchBetMessage shared.BatchBetMessage
	err := batchBetMessage.Deserialize(message.Payload)
	if err != nil {
		sendResponse(s.clientConn, shared.BetResponse(false))
		return
	}

	var successfullBets []*bets.Bet
	var errorCount int

	for _, bet := range batchBetMessage.ReceivedBets {
		number, err := strconv.Atoi(bet[5])
		if err != nil {
			errorCount++
			continue
		}
		bet, err := bets.NewBet(bet[0], bet[1], bet[2], bet[3], bet[4], number)
		if err != nil {
			errorCount++
			continue
		}

		successfullBets = append(successfullBets, bet)
	}

	if errorCount > 0 {
		log.Printf("action: apuesta_recibida | result: fail | cantidad: %v", errorCount)
		sendResponse(s.clientConn, shared.BetResponse(false))
		err = bets.StoreBets(successfullBets)

		if err != nil {
			log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
			sendResponse(s.clientConn, shared.BetResponse(false))
		}

		return
	}

	err = bets.StoreBets(successfullBets)

	if err != nil {
		log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
		sendResponse(s.clientConn, shared.BetResponse(false))
		return
	}

	log.Printf("action: apuesta_recibida | result: success | cantidad: %v", len(successfullBets))

	sendResponse(s.clientConn, shared.BetResponse(true))
}

func sendResponse(conn net.Conn, response shared.BetResponse) error {
	responseSerialized, _ := response.Serialize()
	return shared.WriteSafe(conn, responseSerialized)
}

func (s *Server) handleAllBetsSentMessage() {
	s.receivedAgencies++
	if s.receivedAgencies == s.totalAgencies {
		log.Printf("action: sorteo | result: success")
		s.IdentifyWinners()
		s.receivedAgencies = 0
	}
}

func (s *Server) IdentifyWinners() {
	loadedBets, err := bets.LoadBets()
	if err != nil {
		log.Printf("action: identificar_ganadores | result: fail | error: %v", err)
		return
	}

	winners := make(map[int][]string)
	for _, bet := range loadedBets {
		if bets.HasWon(bet) {
			winners[bet.Agency] = append(winners[bet.Agency], bet.Document)
		}
	}

	s.winners = winners

}

func (s *Server) handleResultsQueryMessage(message *shared.RawMessage) {
	var resultsQueryMessage shared.ResultsQueryMessage
	err := resultsQueryMessage.Deserialize(message.Payload)
	if err != nil {
		log.Printf("action: handle_results_query_message | result: fail | error: %v", err)
		return
	}
	if s.winners == nil {
		message := shared.ResultUnavailableMessage{}
		messageSerialized, _ := message.Serialize()
		err := shared.WriteSafe(s.clientConn, messageSerialized)
		if err != nil {
			log.Printf("action: handle_results_query_message | result: fail | error: %v", err)
		}
		return
	}
	winners := s.winners[resultsQueryMessage.Agency]
	response := shared.ResultsResponseMessage{Winners: winners}
	responseSerialized, _ := response.Serialize()
	shared.WriteSafe(s.clientConn, responseSerialized)
}
