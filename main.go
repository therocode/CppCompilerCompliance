package main

import (
	"context"
	"cppimpbot/compliance"
	"cppimpbot/scraper"
	"cppimpbot/util"
	"database/sql"
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

	//signal that's used to signal quit
	quitChan := make(chan struct{})

	//launch ticker that polls website
	webFetcherTicker := time.NewTicker(30 * time.Second)
	go func() {
		println("starting web fetcher ticker")
		for {
			select {
			case <-webFetcherTicker.C:
				log.Println("fetching cppreference")
				scraped, err := scraper.ScrapeCppSupport()

				if err != nil {
					log.Printf("error when scraping cpp support data: %v\n", err)
				} else {
					//use scraped data
					log.Println("done fetching")

					for _, cppVersion := range scraped.Versions {
						log.Printf("checking for changes in features of C++%v\n", cppVersion.Version)

						for _, feature := range cppVersion.Features {

							dbFeature := compliance.Feature{
								Name:             feature.Name,
								CppVersion:       cppVersion.Version,
								PaperName:        sql.NullString{feature.PaperName, true},
								PaperLink:        sql.NullString{feature.PaperLink, true},
								GccSupport:       feature.GccSupport.HasSupport,
								GccDisplayText:   sql.NullString{feature.GccSupport.DisplayString, true},
								GccExtraText:     sql.NullString{feature.GccSupport.ExtraString, true},
								ClangSupport:     feature.ClangSupport.HasSupport,
								ClangDisplayText: sql.NullString{feature.ClangSupport.DisplayString, true},
								ClangExtraText:   sql.NullString{feature.ClangSupport.ExtraString, true},
								MsvcSupport:      feature.MsvcSupport.HasSupport,
								MsvcDisplayText:  sql.NullString{feature.MsvcSupport.DisplayString, true},
								MsvcExtraText:    sql.NullString{feature.MsvcSupport.ExtraString, true},
							}

							differs, lastEntry, err := complianceStorageService.GetLastIfDiffers(context.Background(), &dbFeature)

							if err != nil {
								log.Printf("Error getting last differing for feature '%v', skipping entry: %v\n", feature.Name, err)
								continue
							}

							if differs && lastEntry == nil { //there was no prior entry, so add the first one
								log.Printf("creating new entry of feature '%v' in database", feature.Name)
								err = complianceStorageService.CreateEntry(context.Background(), &dbFeature)

								if err != nil {
									log.Printf("error creating entry: %v", err)
								}
							} else {
								log.Printf("nothing to be done")
							}
						}
					}
				}
			case <-quitChan:
				println("stopping web fetcher ticker")
				webFetcherTicker.Stop()
				return
			}
		}
	}()

	//pause here until quit yo
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt)
	go func() {
		for _ = range ctrlCChan {
			println("will shut down...")
			close(quitChan)
		}
	}()

	<-quitChan

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
