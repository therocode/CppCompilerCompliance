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

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	_ "github.com/mattn/go-sqlite3"
)

type Configuration struct {
	StorageMode           string
	Database              string
	MigrateDir            string
	ConsumerKey           string
	ConsumerSecret        string
	AccessToken           string
	AccessSecret          string
	MaintainerTwitterId   string
	SafeMode              bool
	SafeModeMaxReports    int
	WebScrapeInterval     int
	TwitterReportInterval int
	SupressReporting      bool //if this is true, all changes will be marked as reported without actually reporting them
	DryReporting          bool //if this is true, changes will be reported using prints only, and not marked as reported
}

var rootCommand = &cobra.Command{
	Use:   "server",
	Short: "Start cpp impl bot service",
	RunE:  rootCmdFunc,
}

var testCommand = &cobra.Command{
	Use:   "test",
	Short: "Test the text reporting functionality",
	RunE:  testCmdFunc,
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
		if err := util.SqliteMigrateUp(cfg.Database, cfg.MigrateDir); err != nil {
			return err
		}

		//create database instance that services will use
		db, err := util.SqliteConnect(cfg.Database)
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

	//set up twitter client
	config := oauth1.NewConfig(cfg.ConsumerKey, cfg.ConsumerSecret)
	token := oauth1.NewToken(cfg.AccessToken, cfg.AccessSecret)
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)
	// Twitter client
	client := twitter.NewClient(httpClient)

	//signal that's used to signal quit
	quitChan := make(chan struct{})

	//launch ticker that polls website
	webFetcherTicker := time.NewTicker(time.Duration(cfg.WebScrapeInterval) * time.Second)
	go func() {
		log.Printf("starting web fetcher ticker with %v seconds interval", cfg.WebScrapeInterval)
		for {
			select {
			case <-webFetcherTicker.C:
				scraped, err := scraper.ScrapeCppSupport()

				if err != nil {
					log.Printf("error when scraping cpp support data: %v\n", err)
				} else {
					for _, cppVersion := range scraped.Versions {
						for _, feature := range cppVersion.Features {

							dbFeature := compliance.Feature{
								Name:             feature.Name,
								CppVersion:       cppVersion.Version,
								PaperName:        sql.NullString{feature.PaperName, true},
								PaperLink:        sql.NullString{feature.PaperLink, true},
								GccSupport:       feature.GccSupport.Support,
								GccDisplayText:   sql.NullString{feature.GccSupport.DisplayString, true},
								GccExtraText:     sql.NullString{feature.GccSupport.ExtraString, true},
								ClangSupport:     feature.ClangSupport.Support,
								ClangDisplayText: sql.NullString{feature.ClangSupport.DisplayString, true},
								ClangExtraText:   sql.NullString{feature.ClangSupport.ExtraString, true},
								MsvcSupport:      feature.MsvcSupport.Support,
								MsvcDisplayText:  sql.NullString{feature.MsvcSupport.DisplayString, true},
								MsvcExtraText:    sql.NullString{feature.MsvcSupport.ExtraString, true},
							}

							differs, lastEntry, err := complianceStorageService.GetLastIfDiffers(context.Background(), &dbFeature)

							if err != nil {
								log.Printf("Error getting last differing for feature '%v', skipping entry: %v\n", feature.Name, err)
								continue
							}

							if differs && lastEntry == nil { //there was no prior entry, so add the first one
								log.Printf("creating new entry of feature '%v' in database because there is no previous one", feature.Name)

								err = complianceStorageService.CreateEntry(context.Background(), &dbFeature)

								if err != nil {
									log.Printf("error creating entry: %v", err)
								}
							} else if differs {
								log.Printf("creating new entry of feature '%v' in database because the old one is different", feature.Name)

								err = complianceStorageService.CreateEntry(context.Background(), &dbFeature)

								if err != nil {
									log.Printf("error creating entry: %v", err)
								}
							} else {
								//log.Printf("nothing to be done")
							}
						}
					}
				}
			case <-quitChan:
				log.Println("stopping web fetcher ticker")
				webFetcherTicker.Stop()
				return
			}
		}
	}()

	//launch ticker that posts reports as tweets
	tweetReporterTicker := time.NewTicker(time.Duration(cfg.TwitterReportInterval) * time.Second)
	go func() {
		log.Printf("starting tweet reporter ticker with %v seconds interval", cfg.TwitterReportInterval)
		for {
			select {
			case <-tweetReporterTicker.C:

				unreportedEntries, err := complianceStorageService.GetNotTwitterReported(context.Background())

				if err != nil {
					log.Printf("error getting entries not reported to twitter: %v\n", err)
					continue
				}

				amountToReport := len(unreportedEntries)

				if amountToReport > cfg.SafeModeMaxReports && cfg.SafeMode {
					log.Printf("Found %v entries to report, this is too many for safe mode (limit is %v)... will not report\n", amountToReport, cfg.SafeModeMaxReports)

					message := fmt.Sprintf("Hello! There were too many reports for safe mode (limit is %v). I won't report anything until you look into this. Amount of reports was %v", cfg.SafeModeMaxReports, amountToReport)
					//directmessage, httpresponse, err
					_, _, err = client.DirectMessages.EventsNew(&twitter.DirectMessageEventsNewParams{
						Event: &twitter.DirectMessageEvent{
							Type: "message_create",
							Message: &twitter.DirectMessageEventMessage{
								Target: &twitter.DirectMessageTarget{
									RecipientID: cfg.MaintainerTwitterId,
								},
								Data: &twitter.DirectMessageData{
									Text: message,
								},
							},
						},
					})

					if err != nil {
						log.Printf("did not manage to report by twitter pm that there are too many reports (%v reports). Errors was: %v\n", amountToReport, err)
					}

					log.Printf("stopping tweet reporter ticker\n")

					return
				}

				for _, entry := range unreportedEntries {
					previous, err := complianceStorageService.GetPreviousFeatureEntry(context.Background(), &entry)

					if err != nil {
						log.Printf("error when getting previous feature entry: %v\n", err)
						continue
					}

					twitterReport, err := compliance.FeatureToTwitterReport(previous, &entry)

					if err != nil {
						log.Printf("not capable of turning update into report. will try to report this as private tweet: %v\n", err)
						if entry.ReportedBroken {
							log.Printf("this error is already reported, skip entry\n")
							continue
						}

						message := fmt.Sprintf("Hello! There was an issue with a change on cppreference that I don't know how to turn into a report.\nThe involved entries are '%v' '%v' and '%v' '%v'. \nFull expansion of those:\n\n%v\n\n%v", previous.Name, previous.Timestamp, entry.Name, entry.Timestamp, previous, entry)
						//directmessage, httpresponse, err
						_, _, err = client.DirectMessages.EventsNew(&twitter.DirectMessageEventsNewParams{
							Event: &twitter.DirectMessageEvent{
								Type: "message_create",
								Message: &twitter.DirectMessageEventMessage{
									Target: &twitter.DirectMessageTarget{
										RecipientID: cfg.MaintainerTwitterId,
									},
									Data: &twitter.DirectMessageData{
										Text: message,
									},
								},
							},
						})

						if err != nil {
							log.Printf("did not manage to report by twitter pm that I couldn't report to twitter: %v\n", err)
						} else {
							log.Printf("error report sent.\n")
							complianceStorageService.SetErrorReported(context.Background(), &entry)
						}
						continue
					}

					if !cfg.SupressReporting {
						messagePrefix := "Dry run: "
						//tweet, resp, err
						if !cfg.DryReporting && twitterReport != "" { //do not post if we do dry run or message is empty
							_, _, err = client.Statuses.Update(twitterReport, nil)
							messagePrefix = ""
						}

						if twitterReport != "" {
							log.Printf(messagePrefix+"posting tweet: %v\n", twitterReport)
						} else {
							log.Printf(messagePrefix + "found change that I don't care about. setting as reported.\n")
						}

						if err != nil {
							log.Printf("error posting tweet update: %v\n", err)
							continue
						} else {
							if !cfg.DryReporting {
								complianceStorageService.SetTwitterReported(context.Background(), &entry)
							}
						}
					} else {
						log.Printf("got twitter report which will be supressed: %v\n", twitterReport)
						complianceStorageService.SetTwitterReported(context.Background(), &entry)
					}
				}
				break
			case <-quitChan:
				log.Println("stopping tweet reporter ticker")
				tweetReporterTicker.Stop()
				return
			}
		}
	}()

	//pause here until quit yo
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt)
	go func() {
		for _ = range ctrlCChan {
			log.Println("will shut down...")
			close(quitChan)
		}
	}()

	<-quitChan

	return nil
}

func testCmdFunc(cmd *cobra.Command, args []string) error {
	log.Print("=====Testing text reports=====\n\n")

	//note: fake data
	baseFeature := compliance.Feature{
		Name:             "Initializer list constructors in class template argument deduction",
		CppVersion:       20,
		PaperName:        sql.NullString{"P0702R1", true},
		PaperLink:        sql.NullString{"https://wg21.link/P0702R1", true},
		GccSupport:       0,
		GccDisplayText:   sql.NullString{"", true},
		GccExtraText:     sql.NullString{"", true},
		ClangSupport:     1,
		ClangDisplayText: sql.NullString{"6 (partial)*", true},
		ClangExtraText:   sql.NullString{"only supported if flag supplied", true},
		MsvcSupport:      0,
		MsvcDisplayText:  sql.NullString{"", true},
		MsvcExtraText:    sql.NullString{"", true},
	}

	baseFeatureSupportsTwo := baseFeature
	baseFeatureSupportsTwo.MsvcSupport = 2
	baseFeatureSupportsTwo.MsvcDisplayText.String = "19.20"
	baseFeatureSupportsTwo.MsvcExtraText.String = "not bug free"

	newSupportFeature := baseFeature
	newSupportFeature.GccSupport = 1
	newSupportFeature.GccDisplayText = sql.NullString{"9*", true}
	newSupportFeature.GccExtraText = sql.NullString{"still some bugs", true}

	newSupportMultipleFeature := newSupportFeature
	newSupportMultipleFeature.MsvcSupport = 1
	newSupportMultipleFeature.MsvcDisplayText = sql.NullString{"19.20", true}
	newSupportMultipleFeature.MsvcExtraText = sql.NullString{"", true}

	textChangeFeature := baseFeatureSupportsTwo
	textChangeFeature.ClangDisplayText = sql.NullString{"6", true}
	textChangeFeature.ClangExtraText = sql.NullString{"", true}

	textChangeMultipleFeature := textChangeFeature
	textChangeMultipleFeature.MsvcDisplayText = sql.NullString{"19.20", true}
	textChangeMultipleFeature.MsvcExtraText = sql.NullString{"one bug", true}

	//test for when a new feature is listed
	text, err := compliance.FeatureToTwitterReport(nil, &baseFeature)

	if err != nil {
		log.Printf("Report when a new feature is added to the listing:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a new feature is added to the listing:\n%v\n\n", text)
	}

	//test for when a new feature is listed with full support
	text, err = compliance.FeatureToTwitterReport(nil, &newSupportMultipleFeature)

	if err != nil {
		log.Printf("Report when a new feature is added to the listing with full support:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a new feature is added to the listing with full support:\n%v\n\n", text)
	}

	//test for when a feature has gained support in a compiler
	text, err = compliance.FeatureToTwitterReport(&baseFeature, &newSupportFeature)

	if err != nil {
		log.Printf("Report when a feature has gained compiler support:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a feature has gained compiler support:\n%v\n\n", text)
	}

	//test for when a feature has gained multiple support in a compiler
	text, err = compliance.FeatureToTwitterReport(&baseFeature, &newSupportMultipleFeature)

	if err != nil {
		log.Printf("Report when a feature has gained multiple compiler support:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a feature has gained multiple compiler support:\n%v\n\n", text)
	}

	//test for when a feature has lost support in a compiler
	text, err = compliance.FeatureToTwitterReport(&newSupportFeature, &baseFeature)

	if err != nil {
		log.Printf("Report when a feature has lost compiler support:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a feature has lost compiler support:\n%v\n\n", text)
	}

	//test for when a feature has lost multiple support in a compiler
	text, err = compliance.FeatureToTwitterReport(&newSupportMultipleFeature, &baseFeature)

	if err != nil {
		log.Printf("Report when a feature has lost multiple compiler support:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a feature has lost multiple compiler support:\n%v\n\n", text)
	}

	//test for when a feature has had its text changed
	text, err = compliance.FeatureToTwitterReport(&baseFeatureSupportsTwo, &textChangeFeature)

	if err != nil {
		log.Printf("Report when a feature had its text changed:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a feature had its text changed:\n%v\n\n", text)
	}

	//test for when a feature has had mutiple texts changed
	text, err = compliance.FeatureToTwitterReport(&baseFeatureSupportsTwo, &textChangeMultipleFeature)

	if err != nil {
		log.Printf("Report when a feature had multiple text changed:\n Error: %v\n\n", err)
	} else {
		log.Printf("Report when a feature had multiple text changed:\n%v\n\n", text)
	}

	return nil
}

func initConfig() {
	//viper.SetDefault("Port", "8080")
	viper.SetDefault("DatabaseConnection", "./data.db")
	viper.SetDefault("MigrateDir", "./migrations")
	viper.SetDefault("StorageMode", "sqlite3")
	viper.SetDefault("SafeMode", true)
	viper.SetDefault("SafeModeMaxReports", 5)
	viper.SetDefault("WebScrapeInterval", 300)
	viper.SetDefault("TwitterReportInterval", 300)
	viper.SetDefault("SupressReporting", false)
	viper.SetDefault("DryReporting", true)

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

	rootCommand.AddCommand(testCommand)

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
