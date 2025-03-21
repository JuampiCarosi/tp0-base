package client

import (
	"net"
	"os"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/comm"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config   ClientConfig
	conn     net.Conn
	shutdown bool
	bet      bets.Bet
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, bet bets.Bet) *Client {
	client := &Client{
		config:   config,
		shutdown: false,
		bet:      bet,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Create the connection the server in every loop iteration. Send an
		if c.shutdown {
			break
		}

		err := c.createClientSocket()
		if err != nil {
			log.Errorf("action: create_client_socket | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		betMessage := comm.BetMessage{
			ReceivedBet: c.bet,
		}
		messageBytes, err := betMessage.Serialize()
		if err != nil {
			log.Errorf("action: serialize_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}
		// log.Debugf("messageBytes: %v, length: %v, binary: %b, string: %v", messageBytes, len(messageBytes), messageBytes, string(messageBytes))
		c.conn.Write(messageBytes)

		response, err := comm.MessageFromSocket(&c.conn)

		if err != nil {
			log.Errorf("action: apuesta enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		if response.Type != comm.BetResponseType {
			log.Errorf("action: apuesta enviada | result: fail | client_id: %v | error: unknown response type %v",
				c.config.ID,
				response.Type,
			)
			return
		}

		var responseMessage comm.BetResponse
		err = responseMessage.Deserialize(response.Payload)
		if err != nil {
			log.Errorf("action: apuesta enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		if responseMessage {
			log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
				c.bet.Document,
				c.bet.Number,
			)
		} else {
			log.Infof("action: apuesta_enviada | result: fail | dni: %v | numero: %v",
				c.bet.Document,
				c.bet.Number,
			)
		}

		time.Sleep(c.config.LoopPeriod)

	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) Cleanup(signal os.Signal) {
	c.shutdown = true
	if c.conn == nil {
		return
	}

	err := c.conn.Close()
	if err != nil {
		log.Infof("action: connection_closed | result: success | client_id: %v | signal: %v | closed resource: %v", c.config.ID, signal, err)
	}
}
