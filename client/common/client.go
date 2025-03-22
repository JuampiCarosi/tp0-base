package common

import (
	"encoding/csv"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/shared"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	MaxAmount     int
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

// SendBatches Send messages to the client until some time threshold is met
func (c *Client) SendBatches() error {
	agencyFile, err := os.Open("/agency.csv")
	if err != nil {
		log.Errorf("action: load_agency_bets | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	defer agencyFile.Close()

	reader := csv.NewReader(agencyFile)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	eof := false
	for !c.shutdown && !eof {
		// Create the connection the server in every loop iteration. Send an
		if c.shutdown {
			break
		}

		batch, err := c.LoadAgencyBatch(reader)
		if err == io.EOF {
			eof = true
			if len(batch) == 0 {
				break
			}
		}
		if err != nil {
			log.Errorf("action: load_agency_batch | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			continue
		}
		err = c.SendBatch(batch)
		if err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			continue
		}
		time.Sleep(c.config.LoopPeriod)

	}
	time.Sleep(c.config.LoopPeriod)

	allBetsSentMessage := shared.AllBetsSentMessage{}
	messageBytes, err := allBetsSentMessage.Serialize()
	if err != nil {
		log.Errorf("action: serialize_finish_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	err = c.createClientSocket()
	if err != nil {
		log.Errorf("action: create_client_socket | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	defer c.conn.Close()
	err = shared.WriteSafe(c.conn, messageBytes)
	if err != nil {
		log.Errorf("action: write_finish_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	log.Infof("action: batches_finished | result: success | client_id: %v", c.config.ID)
	return nil
}

func (c *Client) SendBatch(batch [][]string) error {

	err := c.createClientSocket()
	if err != nil {
		log.Errorf("action: create_client_socket | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	defer c.conn.Close()

	batchMessage := shared.BatchBetMessage{
		ReceivedBets: batch,
	}
	messageBytes, err := batchMessage.Serialize()
	if err != nil {
		log.Errorf("action: serialize_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	err = shared.WriteSafe(c.conn, messageBytes)
	if err != nil {
		log.Errorf("action: write_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	response, err := shared.MessageFromSocket(&c.conn)

	if err != nil {
		log.Errorf("action: batch_sent | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	if response.Type != shared.BetResponseType {
		log.Errorf("action: batch_sent | result: fail | client_id: %v | error: unknown response type %v",
			c.config.ID,
			response.Type,
		)
		return errors.New("unknown response type")
	}

	var responseMessage shared.BetResponse
	err = responseMessage.Deserialize(response.Payload)
	if err != nil {
		log.Errorf("action: batch_sent | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	if responseMessage {
		log.Infof("action: batch_sent | result: success | client_id: %v",
			c.config.ID,
		)
	} else {
		log.Infof("action: batch_sent | result: fail | client_id: %v",
			c.config.ID,
		)
	}
	return nil
}

func (c *Client) SendResultsQuery() error {
	agency, err := strconv.Atoi(c.config.ID)
	if err != nil {
		log.Errorf("action: send_results_query | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	resultsQueryMessage := shared.ResultsQueryMessage{
		Agency: agency,
	}
	messageBytes, err := resultsQueryMessage.Serialize()
	if err != nil {
		log.Errorf("action: serialize_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}

	for i := 10; i > 0; i-- {
		err = c.createClientSocket()
		if err != nil {
			log.Errorf("action: create_client_socket | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return err
		}
		defer c.conn.Close()
		err = shared.WriteSafe(c.conn, messageBytes)
		if err != nil {
			log.Errorf("action: write_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return err
		}

		response, err := shared.MessageFromSocket(&c.conn)
		if err != nil {
			log.Errorf("action: send_results_query | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return err
		}

		switch response.Type {
		case shared.ResultsResponseType:
			var resultsResponseMessage shared.ResultsResponseMessage
			err = resultsResponseMessage.Deserialize(response.Payload)
			if err != nil {
				log.Errorf("action: send_results_query | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
				return err
			}
			log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
				len(resultsResponseMessage.Winners),
			)
			return nil
		case shared.ResultUnavailableType:

			time.Sleep(time.Millisecond * 100)
		default:
			log.Errorf("action: consulta_ganadores | result: fail | client_id: %v | error: unknown response type %v",
				c.config.ID,
				response.Type,
			)
			return errors.New("unknown response type")
		}
	}
	return nil
}
func (c *Client) LoadAgencyBatch(reader *csv.Reader) ([][]string, error) {

	var loadedBets [][]string

	for i := 0; i < c.config.MaxAmount; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			return loadedBets, err
		}
		if err != nil {
			log.Errorf("action: load_agency_bets | result: fail | client_id: %v | error: %v",
				c.config.ID, err)
			return nil, err
		}

		recordWithAgency := append([]string{c.config.ID}, record...)

		loadedBets = append(loadedBets, recordWithAgency)
	}

	return loadedBets, nil

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
