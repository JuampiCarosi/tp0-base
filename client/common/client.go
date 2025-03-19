package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

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

type Bet struct {
	Name         string
	SurName      string
	Document     string
	BirthDate    time.Time
	BettedNumber int
}

// Client Entity that encapsulates how
type Client struct {
	config   ClientConfig
	conn     net.Conn
	shutdown bool
	bet      Bet
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, bet Bet) *Client {
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
		bet := []byte(fmt.Sprintf("%v;%v;%v;%v;%v;%v", c.config.ID, c.bet.Name, c.bet.SurName, c.bet.Document, c.bet.BirthDate.Format("2006-01-02"), c.bet.BettedNumber))
		lenBytes := []byte(fmt.Sprintf("%v ", len(bet)))
		message := append(lenBytes, bet...)

		written, err := c.conn.Write(message)
		for written < len(message) {
			written, err = c.conn.Write(message[written:])
		}

		if err != nil {
			log.Errorf("action: send_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.conn.Close()

		if err != nil {
			log.Errorf("action: apuesta enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		if msg != "OK\n" {
			log.Errorf("action: apuesta enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				msg,
			)
			return
		}

		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
			c.bet.Document,
			c.bet.BettedNumber,
		)
		// Wait a time between sending one message and the next one
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
