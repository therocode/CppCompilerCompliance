package main

import (
	"context"
	"cppimpbot/compliance"
	"cppimpbot/util"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/mattn/go-sqlite3"
)

type Configuration struct {
	Port               string
	DatabaseConnection string
	MigrateDir         string
	StorageMode        string
}

var rootCommand = &cobra.Command{
	Use:   "server",
	Short: "Start cpp impl bot service",
	RunE:  rootCmdFunc,
}

func rootCmdFunc(cmd *cobra.Command, args []string) error {

	cfg := &Configuration{}

	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	//services
	var complianceStorageService compliance.Service

	//initialise services
	switch cfg.StorageMode {
	case "sqlite3":
		//database migration
		if err := util.SqliteMigrateUp(cfg.DatabaseConnection, cfg.MigrateDir); err != nil {
			return err
		}

		//create database instance that services will use
		db, err := util.SqliteConnect(cfg.DatabaseConnection)
		if err != nil {
			return err
		}

		complianceStorageService = compliance.NewSqliteService(db)
	case "dummy":
		//complianceStorageService = dog.NewDummySerbice(db)
	default:
		return fmt.Errorf("Invalid storageMode: %s", cfg.StorageMode)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		complianceStorageService.Close(ctx)
		cancel()
	}()

	//managers
	//dogManager := dog.NewManager(complianceStorageService)

	//pause here until quit yo
	quit := make(chan struct{})
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt)
	go func() {
		for _ = range ctrlCChan {
			println("will shut down...")
			quit <- struct{}{}
		}
	}()

	<-quit

	println("shut down gracefully")

	return nil
}

func initConfig() {
	//viper.SetDefault("Port", "8080")
	viper.SetDefault("DatabaseConnection", "./data.db")
	viper.SetDefault("MigrateDir", "./migrations")
	viper.SetDefault("StorageMode", "sqlite3")

	var cfgFile string

	rootCommand.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config.toml)")

	failOnMissingConfig := false
	if cfgFile != "" {
		failOnMissingConfig = true
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	err := viper.ReadInConfig()            //find and read the config file
	if failOnMissingConfig && err != nil { // Handle errors reading the config file
		log.Fatal("Failed to read config", err)
	}
}

func main() {
	cobra.OnInitialize(initConfig)

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
