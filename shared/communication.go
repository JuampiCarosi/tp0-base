package shared

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
)

type MessageType int32

const (
	BetType MessageType = iota
	BetResponseType
	BatchBetType
	AllBetsSentType
	ResultsQueryType
	ResultUnavailableType
	ResultsResponseType
)

type Message interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

type BetMessage struct {
	Message
	ReceivedBet bets.Bet
}

func (m *BetMessage) Serialize() ([]byte, error) {
	payload := []byte(fmt.Sprintf("%v;%v;%v;%v;%v;%v", m.ReceivedBet.Agency, m.ReceivedBet.FirstName, m.ReceivedBet.LastName, m.ReceivedBet.Document, m.ReceivedBet.BirthDate.Format("2006-01-02"), m.ReceivedBet.Number))
	buffer := bytes.NewBuffer([]byte{})

	binary.Write(buffer, binary.BigEndian, uint32(BetType))
	binary.Write(buffer, binary.BigEndian, uint32(len(payload)))

	buffer.Write(payload)
	return buffer.Bytes(), nil
}

func (m *BetMessage) Deserialize(data string) error {
	parts := strings.Split(data, ";")
	number, err := strconv.Atoi(parts[5])
	if err != nil {
		return err
	}
	bet, err := bets.NewBet(parts[0], parts[1], parts[2], parts[3], parts[4], number)
	if err != nil {
		return err
	}

	m.ReceivedBet = *bet

	return nil
}

type BetResponse bool

func (m *BetResponse) Serialize() ([]byte, error) {
	var payload string
	if *m {
		payload = "SUCCESS"
	} else {
		payload = "ERROR"
	}

	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint32(BetResponseType))
	binary.Write(buffer, binary.BigEndian, uint32(len(payload)))
	buffer.Write([]byte(payload))

	return buffer.Bytes(), nil
}

func (m *BetResponse) Deserialize(data string) error {
	if data == "SUCCESS" {
		*m = true
	} else {
		*m = false
	}

	return nil
}

type BatchBetMessage struct {
	Message
	ReceivedBets [][]string
}

func (m *BatchBetMessage) Serialize() ([]byte, error) {
	var payloadString []string
	for _, bet := range m.ReceivedBets {
		payloadString = append(payloadString, fmt.Sprintf("%v;%v;%v;%v;%v;%v", bet[0], bet[1], bet[2], bet[3], bet[4], bet[5]))
	}

	buffer := bytes.NewBuffer([]byte{})
	payload := []byte(strings.Join(payloadString, "\n"))

	binary.Write(buffer, binary.BigEndian, uint32(BatchBetType))
	binary.Write(buffer, binary.BigEndian, uint32(len(payload)))

	buffer.Write(payload)
	return buffer.Bytes(), nil
}

func (m *BatchBetMessage) Deserialize(data string) error {
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		parts := strings.Split(line, ";")
		m.ReceivedBets = append(m.ReceivedBets, parts)
	}

	return nil
}

type RawMessage struct {
	Type    MessageType
	Length  int
	Payload string
}

func MessageFromSocket(socket *net.Conn) (*RawMessage, error) {
	reader := bufio.NewReader(*socket)
	u8Buffer := make([]byte, 4)
	_, err := io.ReadFull(reader, u8Buffer)
	if err != nil {
		return nil, err
	}
	messageType := binary.BigEndian.Uint32(u8Buffer)
	_, err = io.ReadFull(reader, u8Buffer)
	if err != nil {
		return nil, err
	}
	messageLength := binary.BigEndian.Uint32(u8Buffer)
	payload := make([]byte, messageLength)
	_, err = io.ReadFull(reader, payload)
	if err != nil {
		return nil, err
	}

	return &RawMessage{
		Type:    MessageType(messageType),
		Length:  int(messageLength),
		Payload: string(payload),
	}, nil
}

type AllBetsSentMessage struct {
	Message
	Agency int
}

func (m *AllBetsSentMessage) GetMessageType() MessageType {
	return AllBetsSentType
}

func (m *AllBetsSentMessage) Serialize() ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint32(AllBetsSentType))
	binary.Write(buffer, binary.BigEndian, uint32(4))
	binary.Write(buffer, binary.BigEndian, uint32(m.Agency))
	return buffer.Bytes(), nil
}

func (m *AllBetsSentMessage) Deserialize(data string) error {
	agency := binary.BigEndian.Uint32([]byte(data))
	m.Agency = int(agency)
	return nil
}

type ResultsQueryMessage struct {
	Message
	Agency int
}

func (m *ResultsQueryMessage) GetMessageType() MessageType {
	return ResultsQueryType
}

func (m *ResultsQueryMessage) Serialize() ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint32(ResultsQueryType))
	binary.Write(buffer, binary.BigEndian, uint32(4))
	binary.Write(buffer, binary.BigEndian, uint32(m.Agency))
	return buffer.Bytes(), nil
}

func (m *ResultsQueryMessage) Deserialize(data string) error {
	agency := binary.BigEndian.Uint32([]byte(data))
	m.Agency = int(agency)
	return nil
}

type ResultUnavailableMessage struct {
	Message
}

func (m *ResultUnavailableMessage) GetMessageType() MessageType {
	return ResultUnavailableType
}

func (m *ResultUnavailableMessage) Serialize() ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint32(ResultUnavailableType))
	binary.Write(buffer, binary.BigEndian, uint32(0))
	return buffer.Bytes(), nil
}

func (m *ResultUnavailableMessage) Deserialize(data string) error {
	return nil
}

type ResultsResponseMessage struct {
	Message
	Winners []string
}

func (m *ResultsResponseMessage) GetMessageType() MessageType {
	return ResultsResponseType
}

func (m *ResultsResponseMessage) Serialize() ([]byte, error) {
	payload := strings.Join(m.Winners, ";")
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint32(ResultsResponseType))
	binary.Write(buffer, binary.BigEndian, uint32(len(payload)))
	buffer.Write([]byte(payload))
	return buffer.Bytes(), nil
}

func (m *ResultsResponseMessage) Deserialize(data string) error {
	if data == "" {
		m.Winners = []string{}
		return nil
	}
	parts := strings.Split(data, ";")
	m.Winners = parts
	return nil
}
