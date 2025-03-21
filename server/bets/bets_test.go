package bets

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBetInitMustKeepFields(t *testing.T) {
	bet, err := NewBet("1", "first", "last", "10000000", "2000-12-20", 7500)
	assert.NoError(t, err)
	assert.Equal(t, 1, bet.Agency)
	assert.Equal(t, "first", bet.FirstName)
	assert.Equal(t, "last", bet.LastName)
	assert.Equal(t, "10000000", bet.Document)
	assert.Equal(t, "2000-12-20", bet.BirthDate.Format("2006-01-02"))
	assert.Equal(t, 7500, bet.Number)
}

func TestHasWonWithWinnerNumberMustBeTrue(t *testing.T) {
	bet, err := NewBet("1", "first", "last", "10000000", "2000-12-20", LOTTERY_WINNER_NUMBER)
	assert.NoError(t, err)
	assert.True(t, HasWon(bet))
}

func TestHasWonWithNonWinnerNumberMustBeFalse(t *testing.T) {
	bet, err := NewBet("1", "first", "last", "10000000", "2000-12-20", LOTTERY_WINNER_NUMBER+1)
	assert.NoError(t, err)
	assert.False(t, HasWon(bet))
}

func TestStoreBetsAndLoadBetsKeepsFieldsData(t *testing.T) {
	// Clean up any existing file
	os.Remove(STORAGE_FILEPATH)

	bet, err := NewBet("1", "first", "last", "10000000", "2000-12-20", 7500)
	assert.NoError(t, err)

	toStore := []Bet{*bet}
	err = StoreBets(toStore)
	assert.NoError(t, err)

	fromLoad, err := LoadBets()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fromLoad))
	assertEqualBets(t, &toStore[0], fromLoad[0])
}

func TestStoreBetsAndLoadBetsKeepsRegistryOrder(t *testing.T) {
	// Clean up any existing file
	os.Remove(STORAGE_FILEPATH)

	bet1, err := NewBet("0", "first_0", "last_0", "10000000", "2000-12-20", 7500)
	assert.NoError(t, err)
	bet2, err := NewBet("1", "first_1", "last_1", "10000001", "2000-12-21", 7501)
	assert.NoError(t, err)

	toStore := []Bet{*bet1, *bet2}
	err = StoreBets(toStore)
	assert.NoError(t, err)

	fromLoad, err := LoadBets()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fromLoad))
	assertEqualBets(t, &toStore[0], fromLoad[0])
	assertEqualBets(t, &toStore[1], fromLoad[1])
}

func assertEqualBets(t *testing.T, b1, b2 *Bet) {
	assert.Equal(t, b1.Agency, b2.Agency)
	assert.Equal(t, b1.FirstName, b2.FirstName)
	assert.Equal(t, b1.LastName, b2.LastName)
	assert.Equal(t, b1.Document, b2.Document)
	assert.Equal(t, b1.BirthDate, b2.BirthDate)
	assert.Equal(t, b1.Number, b2.Number)
}
