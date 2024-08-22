package main

import (
	"context"
	"github.com/benyaa/virtual-printer-process-engine/config"
	"github.com/benyaa/virtual-printer-process-engine/engine"
	"github.com/benyaa/virtual-printer-process-engine/osutils"
	"github.com/benyaa/virtual-printer-process-engine/printer"
	"github.com/benyaa/virtual-printer-process-engine/repo"
	"github.com/benyaa/virtual-printer-process-engine/utils"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"github.com/natefinch/lumberjack"
	"github.com/ncruces/zenity"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
)

func main() {
	runAsAService()
}

var cancel context.CancelFunc
var printerCreator printer.Creator
var configLocation = "./config.yaml"

func runAsAService() {
	log.Infof("Initiating...")
	conf := getConfig()
	log.Infof("loaded config successfully")
	setupLogging(conf)

	log.Debugf("Log file set to %s", conf.Logs.Filename)
	assertAdmin()

	var err error
	conf.Workdir, err = utils.EvaluateExpression(conf.Workdir, map[string]interface{}{})
	if err != nil {
		log.WithError(err).Fatalf("Error evaluating workdir")
	}
	createDirs(conf.Workdir, path.Join(conf.Workdir, "contents"), path.Join(conf.Workdir, "jobs"), path.Join(conf.Workdir, "wal"))

	log.Debugf("Output path created")
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	printerCreator = createPrinter(ctx, conf, path.Join(conf.Workdir, "jobs"))
	log.Infof("settuing up write ahead logger")
	writeAheadLogger := repo.NewWriteAheadLogger(path.Join(conf.Workdir, "wal", "wal.log"), conf.WriteAheadLogging)
	log.Info("setting up engine")
	e := engine.New(ctx, conf, printerCreator.GetChannel(), writeAheadLogger)
	log.Info("starting engine")
	go e.Run()

	systray.Run(onReady, onExit)
	log.Debugf("exiting")
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Virtual Printer Process Engine")
	systray.SetTooltip("Virtual Printer Process Engine")
	isRunningAtStartup, err := osutils.IsRunningAtStartup()
	if err != nil {
		log.WithError(err).Errorf("error checking if running at startup")
	}
	mRunAtStartup := systray.AddMenuItemCheckbox("Run at startup", "Run at startup", isRunningAtStartup)
	mQuit := systray.AddMenuItem("Quit", "Quit")
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-mRunAtStartup.ClickedCh:
				log.Debugf("run at startup clicked")
				if !mRunAtStartup.Checked() {
					log.Debugf("checking run at startup")
					mRunAtStartup.Check()
					log.Debugf("running at startup")
					err := osutils.RunAtStartup(configLocation)
					if err != nil {
						log.WithError(err).Errorf("Error running at startup")
						displayErrorMessage("Error", "Error running at startup")
					}
					log.Debugf("set run at startup")
				} else {
					log.Debugf("unchecking run at startup")
					mRunAtStartup.Uncheck()
					log.Debugf("removing from startup")
					err := osutils.RemoveFromStartup()
					if err != nil {
						log.WithError(err).Errorf("error removing from startup")
						displayErrorMessage("Error", "Failed to remove from startup")
					}
					log.Debugf("removed from startup")
				}
			}
		}
	}()
}

func onExit() {
	cancel()
	if printerCreator != nil {
		log.Debugf("Removing virtual printer")
		err := printerCreator.RemoveVirtualPrinter()
		if err != nil {
			log.WithError(err).Errorf("Error removing virtual printer")
		}
	}
	log.Infof("Shutting down...")
}

func setupLogging(conf config.Config) {
	log.Infof("Settings log file as %s", conf.Logs.Filename)
	log.SetOutput(&lumberjack.Logger{
		Filename:   conf.Logs.Filename,
		MaxSize:    conf.Logs.MaxSizeMB,
		MaxBackups: conf.Logs.MaxBackups,
		MaxAge:     conf.Logs.MaxAgeDays,
		Compress:   true,
	})
	log.SetReportCaller(true)
	parsedLevel, err := log.ParseLevel(conf.Logs.Level)
	if err != nil {
		log.WithError(err).Fatal("could not parse log level")
	}
	log.SetLevel(parsedLevel)
	log.SetFormatter(&log.JSONFormatter{})
}

func getConfig() config.Config {
	var argConfigLocation string
	if len(os.Args) > 1 {
		argConfigLocation = os.Args[1]
	}
	if argConfigLocation != "" {
		log.Debugf("Using config location from argument: %s", argConfigLocation)
		configLocation = argConfigLocation
	}
	conf, err := config.ParseConfig(configLocation)
	if err != nil {
		displayErrorMessage("Error", "Could not parse config: "+err.Error())
		log.WithError(err).Fatal(context.Background(), "could not parse config")
	}
	return conf
}

func createPrinter(ctx context.Context, conf config.Config, jobsDir string) printer.Creator {
	printerCreator := printer.Create(ctx, conf, jobsDir)
	err := printerCreator.CreateVirtualPrinter()
	if err != nil {
		log.WithError(err).Errorf("failed to create virtual printer")
		panic(err)
	}
	return printerCreator
}

func assertAdmin() {
	if runtime.GOOS != "windows" {
		log.Debugf("not running on Windows, skipping admin check")
		return
	}
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		log.WithError(err).Errorf("failed to check admin permissions")
		displayErrorMessage("Error", "Error checking admin permissions")
		panic(err)
	}
	if !isAdmin {
		log.Errorf("not running as admin")
		displayErrorMessage("Error", "Not running as admin")
		panic("Not running as admin")
	}
	log.Infof("Running as admin")
}

func displayErrorMessage(title string, message string) {
	err := zenity.Error(message, zenity.Title(title))
	if err != nil {
		log.WithError(err).Errorf("failed to display error message")
	}
}

func createDirs(dirs ...string) {
	for _, dir := range dirs {
		log.Debugf("creating directory %s", dir)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.WithError(err).Fatalf("failed to create directory %s", dir)
		}
	}
}
