package bets

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

const STORAGE_FILEPATH = "./bets.csv"
const LOTTERY_WINNER_NUMBER = 7574

type Bet struct {
	Agency    int
	FirstName string
	LastName  string
	Document  string
	BirthDate time.Time
	Number    int
}

func NewBet(agencyStr string, firstName string, lastName string, document string, birthDateStr string, number int) (*Bet, error) {
	agency, err := strconv.Atoi(agencyStr)
	if err != nil {
		return nil, fmt.Errorf("error converting agency to int: %v", err)
	}
	birthDate, err := time.Parse("2006-01-02", birthDateStr)
	if err != nil {
		return nil, fmt.Errorf("error converting birthDate to time.Time: %v", err)
	}

	return &Bet{
		Agency:    agency,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		BirthDate: birthDate,
		Number:    number,
	}, nil
}

func HasWon(bet *Bet) bool {
	return bet.Number == LOTTERY_WINNER_NUMBER
}

func StoreBets(bets []*Bet) error {
	file, err := os.OpenFile(STORAGE_FILEPATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, bet := range bets {
		record := []string{
			strconv.Itoa(bet.Agency),
			bet.FirstName,
			bet.LastName,
			bet.Document,
			bet.BirthDate.Format("2006-01-02"),
			strconv.Itoa(bet.Number),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing record: %v", err)
		}
	}

	return nil
}

func LoadBets() ([]*Bet, error) {
	file, err := os.Open(STORAGE_FILEPATH)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading records: %v", err)
	}

	bets := make([]*Bet, 0, len(records))
	for _, record := range records {
		number, err := strconv.Atoi(record[5])
		if err != nil {
			return nil, fmt.Errorf("error converting number to int: %v", err)
		}

		bet, err := NewBet(
			record[0],
			record[1],
			record[2],
			record[3],
			record[4],
			number,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating bet: %v", err)
		}
		bets = append(bets, bet)
	}
	return bets, nil
}
