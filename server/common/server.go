package common

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/shared"
)

type Server struct {
	serverSocket     net.Listener
	running          bool
	totalAgencies    int
	receivedAgencies chan int
	winners          map[int][]string
	connections      map[string]net.Conn
	connectionsMutex sync.Mutex
	betsMutex        sync.Mutex
	winnersMutex     sync.Mutex
	wg               sync.WaitGroup
}

func NewServer(address string, agenciesAmount int) (*Server, error) {
	server := &Server{
		running:          true,
		totalAgencies:    agenciesAmount,
		receivedAgencies: make(chan int),
		connections:      make(map[string]net.Conn),
		connectionsMutex: sync.Mutex{},
		betsMutex:        sync.Mutex{},
		winnersMutex:     sync.Mutex{},
		wg:               sync.WaitGroup{},
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("error creating server socket: %v", err)
	}
	server.serverSocket = listener

	return server, nil
}

func (s *Server) Run() {
	defer s.wg.Done()

	s.wg.Add(1)
	go s.identifyWinners()
	for s.running {
		clientConn, err := s.acceptNewConnection()
		if err != nil {
			log.Printf("action: accept_connections | result: failed | error: %v", err)
			return
		}
		s.connectionsMutex.Lock()
		s.connections[clientConn.RemoteAddr().String()] = clientConn
		s.connectionsMutex.Unlock()
		s.wg.Add(1)
		go s.handleClientConnection(clientConn)
	}
}

func (s *Server) Shutdown() {
	s.running = false
	s.connectionsMutex.Lock()
	defer s.connectionsMutex.Unlock()
	for _, conn := range s.connections {
		conn.Close()
		log.Printf("action: connection_closed | result: success | connection: %v", conn.LocalAddr())
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

func (s *Server) handleClientConnection(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		s.connectionsMutex.Lock()
		delete(s.connections, clientConn.RemoteAddr().String())
		s.connectionsMutex.Unlock()
	}()
	defer s.wg.Done()

	errorResponse := shared.BetResponse(false)
	errorResponseSerialized, err := errorResponse.Serialize()

	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		return
	}

	messageType, err := shared.MessageFromSocket(&clientConn)
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		shared.WriteSafe(clientConn, errorResponseSerialized)
		return
	}

	switch messageType.Type {
	case shared.BetType:
		s.handleBetMessage(messageType, clientConn)
	case shared.BatchBetType:
		s.handleBatchBetMessage(messageType, clientConn)
	case shared.AllBetsSentType:
		s.handleAllBetsSentMessage(messageType)
	case shared.ResultsQueryType:
		s.handleResultsQueryMessage(messageType, clientConn)
	default:
		log.Printf("action: handle_client_connection | result: fail | error: unknown message type %v", messageType.Type)
		shared.WriteSafe(clientConn, errorResponseSerialized)
		return
	}

}

func (s *Server) handleBetMessage(message *shared.RawMessage, clientConn net.Conn) {

	var betMessage shared.BetMessage
	err := betMessage.Deserialize(message.Payload)
	if err != nil {
		log.Printf("action: handle_client_connection | result: fail | error: %v", err)
		sendResponse(clientConn, shared.BetResponse(false))
		return
	}
	bet := betMessage.ReceivedBet
	s.betsMutex.Lock()
	err = bets.StoreBets([]*bets.Bet{&bet})
	s.betsMutex.Unlock()

	if err != nil {
		log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
		sendResponse(clientConn, shared.BetResponse(false))
		return
	}

	log.Printf("action: apuesta_almacenada | result: success | dni: %v | numero: %v", bet.Document, bet.Number)
	sendResponse(clientConn, shared.BetResponse(true))
}

func (s *Server) handleBatchBetMessage(message *shared.RawMessage, clientConn net.Conn) {

	var batchBetMessage shared.BatchBetMessage
	err := batchBetMessage.Deserialize(message.Payload)
	if err != nil {
		sendResponse(clientConn, shared.BetResponse(false))
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
		sendResponse(clientConn, shared.BetResponse(false))
		err = bets.StoreBets(successfullBets)

		if err != nil {
			log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
			sendResponse(clientConn, shared.BetResponse(false))
		}

		return
	}

	s.betsMutex.Lock()
	err = bets.StoreBets(successfullBets)
	s.betsMutex.Unlock()

	if err != nil {
		log.Printf("action: apuesta_almacenada | result: fail | error: %v", err)
		sendResponse(clientConn, shared.BetResponse(false))
		return
	}

	log.Printf("action: apuesta_recibida | result: success | cantidad: %v", len(successfullBets))

	sendResponse(clientConn, shared.BetResponse(true))
}

func sendResponse(conn net.Conn, response shared.BetResponse) error {
	responseSerialized, _ := response.Serialize()
	return shared.WriteSafe(conn, responseSerialized)
}

func (s *Server) handleAllBetsSentMessage(message *shared.RawMessage) {
	var allBetsSentMessage shared.AllBetsSentMessage
	err := allBetsSentMessage.Deserialize(message.Payload)
	if err != nil {
		log.Printf("action: handle_all_bets_sent_message | result: fail | error: %v", err)
		return
	}
	s.receivedAgencies <- allBetsSentMessage.Agency
}

func (s *Server) identifyWinners() {
	defer s.wg.Done()
	receivedAgencies := make(map[int]bool)
	for len(receivedAgencies) < s.totalAgencies {
		agency := <-s.receivedAgencies
		receivedAgencies[agency] = true
	}
	s.betsMutex.Lock()
	loadedBets, err := bets.LoadBets()
	s.betsMutex.Unlock()
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

	s.winnersMutex.Lock()
	s.winners = winners
	s.winnersMutex.Unlock()

}

func (s *Server) handleResultsQueryMessage(message *shared.RawMessage, clientConn net.Conn) {
	var resultsQueryMessage shared.ResultsQueryMessage
	err := resultsQueryMessage.Deserialize(message.Payload)
	if err != nil {
		log.Printf("action: handle_results_query_message | result: fail | error: %v", err)
		return
	}
	s.winnersMutex.Lock()
	if s.winners == nil {
		message := shared.ResultUnavailableMessage{}
		messageSerialized, _ := message.Serialize()
		err := shared.WriteSafe(clientConn, messageSerialized)
		if err != nil {
			log.Printf("action: handle_results_query_message | result: fail | error: %v", err)
		}
		return
	}
	winners := s.winners[resultsQueryMessage.Agency]
	response := shared.ResultsResponseMessage{Winners: winners}
	responseSerialized, _ := response.Serialize()
	shared.WriteSafe(clientConn, responseSerialized)
	s.winnersMutex.Unlock()
}
