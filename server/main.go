package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/common"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

type Config struct {
	Port           int
	Ip             string
	LoggingLevel   string
	AgenciesAmount int
}

var log = logging.MustGetLogger("log")

func InitConfig() (*Config, error) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Add env variables supported
	v.BindEnv("default.server_port", "SERVER_PORT")
	v.BindEnv("default.server_ip", "SERVER_IP")
	v.BindEnv("default.logging_level", "LOGGING_LEVEL")
	v.BindEnv("agencies_amount")
	// Try to read configuration from config file. If config file
	// does not exists then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigFile("./config.ini")
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Configuration could not be read from config file. Using env variables instead")
	}

	config := &Config{
		Port:           v.GetInt("default.server_port"),
		Ip:             v.GetString("default.server_ip"),
		LoggingLevel:   v.GetString("default.logging_level"),
		AgenciesAmount: v.GetInt("agencies_amount"),
	}

	if config.Port == 0 {
		return nil, fmt.Errorf("port is not set")
	}

	if config.LoggingLevel == "" {
		return nil, fmt.Errorf("logging_level is not set")
	}

	return config, nil
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
func PrintConfig(config *Config) {
	log.Infof("action: config | result: success | port: %v | listen_backlog: os_default | logging_level: %s | agencies_amount: %v",
		config.Port,
		config.LoggingLevel,
		config.AgenciesAmount,
	)
}

func gracefulShutdown(s *common.Server, wg *sync.WaitGroup) {
	defer wg.Done()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit
	s.Shutdown()
}

func main() {
	config, err := InitConfig()
	if err != nil {
		log.Errorf("error initializing config: %v", err)

	}

	PrintConfig(config)

	if err := InitLogger(config.LoggingLevel); err != nil {
		log.Errorf("error initializing logger: %v", err)
		return

	}

	server, err := common.NewServer(fmt.Sprintf("%s:%d", config.Ip, config.Port), config.AgenciesAmount)
	if err != nil {
		log.Errorf("error initializing server: %v", err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go gracefulShutdown(server, &wg)
	server.Run()

	wg.Wait()

}
