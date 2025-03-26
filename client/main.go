package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/bets"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

// InitConfig Function that uses viper library to parse configuration parameters.
// Viper is configured to read variables from both environment variables and the
// config file ./config.yaml. Environment variables takes precedence over parameters
// defined in the configuration file. If some of the variables cannot be parsed,
// an error is returned
func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("cli")
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Add env variables supported
	v.BindEnv("id")
	v.BindEnv("server", "address")
	v.BindEnv("loop", "period")
	v.BindEnv("loop", "amount")
	v.BindEnv("log", "level")
	v.BindEnv("nombre")
	v.BindEnv("apellido")
	v.BindEnv("documento")
	v.BindEnv("nacimiento")
	v.BindEnv("numero")
	v.BindEnv("batch", "maxAmount")

	v.SetDefault("batch.maxAmount", 105)
	// Try to read configuration from config file. If config file
	// does not exists then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Configuration could not be read from config file. Using env variables instead")
	}

	// Parse time.Duration variables and return an error if those variables cannot be parsed

	if _, err := time.ParseDuration(v.GetString("loop.period")); err != nil {
		return nil, errors.Wrapf(err, "Could not parse CLI_LOOP_PERIOD env var as time.Duration.")
	}

	return v, nil
}

// InitLogger Receives the log level to be set in go-logging as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func InitLogger(logLevel string) error {
	baseBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05} %{level:.5s}     %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(baseBackend, format)

	backendLeveled := logging.AddModuleLevel(backendFormatter)
	logLevelCode, err := logging.LogLevel(logLevel)
	if err != nil {
		return err
	}
	backendLeveled.SetLevel(logLevelCode, "")

	// Set the backends to be used.
	logging.SetBackend(backendLeveled)
	return nil
}

// PrintConfig Print all the configuration parameters of the program.
// For debugging purposes only
func PrintConfig(v *viper.Viper) {
	log.Infof("action: config | result: success | client_id: %s | server_address: %s | loop_amount: %v | loop_period: %v | log_level: %s | nombre: %s | apellido: %s | documento: %s | nacimiento: %v | numero: %v | batch_max_amount: %v",
		v.GetString("id"),
		v.GetString("server.address"),
		v.GetInt("loop.amount"),
		v.GetDuration("loop.period"),
		v.GetString("log.level"),
		v.GetString("nombre"),
		v.GetString("apellido"),
		v.GetString("documento"),
		v.GetTime("nacimiento"),
		v.GetInt("numero"),
		v.GetInt("batch.maxAmount"),
	)
}

func gracefulShutdown(c *common.Client, finished chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	var reason string
	select {
	case s := <-quit:
		reason = s.String()
		log.Infof("action: graceful_shutdown | result: success | reason: %s", reason)
	case <-finished:
		reason = "client finished"
		log.Infof("action: graceful_shutdown | result: timeout | reason: %s", reason)
	}

	c.Cleanup(reason)
}
func main() {
	v, err := InitConfig()
	if err != nil {
		log.Criticalf("%s", err)
	}

	if err := InitLogger(v.GetString("log.level")); err != nil {
		log.Criticalf("%s", err)
	}

	// Print program config with debugging purposes
	PrintConfig(v)

	clientConfig := common.ClientConfig{
		ServerAddress: v.GetString("server.address"),
		ID:            v.GetString("id"),
		LoopAmount:    v.GetInt("loop.amount"),
		LoopPeriod:    v.GetDuration("loop.period"),
		MaxAmount:     v.GetInt("batch.maxAmount"),
	}

	bet := bets.Bet{
		FirstName: v.GetString("nombre"),
		LastName:  v.GetString("apellido"),
		Document:  v.GetString("documento"),
		BirthDate: v.GetTime("nacimiento"),
		Number:    v.GetInt("numero"),
	}

	client := common.NewClient(clientConfig, bet)
	wg := sync.WaitGroup{}
	wg.Add(1)

	finished := make(chan bool)
	go gracefulShutdown(client, finished, &wg)

	client.SendBatches()

	client.SendResultsQuery()

	if !client.Shutdown {
		finished <- true
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}
