package main

import (
	"fmt"
	"ftpdts/src/webserver"
	"github.com/starshiptroopers/ftpdt"
	"github.com/starshiptroopers/ftpdt/datastorage"
	"github.com/starshiptroopers/ftpdt/tmplstorage"
	"github.com/starshiptroopers/uidgenerator"
	"goftp.io/server/core"
	"os"
	"os/signal"
	"time"
)

func main() {

	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	loggerFtp := logInit(config.Logs.Ftp, !config.Logs.FtpNoConsole)
	loggerHttp := logInit(config.Logs.Http, !config.Logs.HttpNoConsole)
	logger := logInit(config.Logs.Ftpdts, !config.Logs.FtpdtsNoConsole)

	ts := tmplstorage.New(config.Templates.Path)

	datastorage.DefaultCacheTTL = time.Second * time.Duration(config.Cache.DataTTL)
	ds := datastorage.NewMemoryDataStorage()

	ug := uidgenerator.New(
		&uidgenerator.Cfg{
			Alfa:      "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
			Format:    "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
			Validator: "[0-9a-zA-Z]{32}",
		},
	)

	ftpOpts := &core.ServerOpts{
		Port:     int(config.Ftp.Port),
		Hostname: config.Ftp.Host,
	}

	ftpd := ftpdt.New(
		&ftpdt.Opts{
			FtpOpts:         ftpOpts,
			TemplateStorage: ts,
			DataStorage:     ds,
			UidGenerator:    ug,
			LogWriter:       loggerFtp.Writer(),
			LogFtpDebug:     false,
		},
	)

	webServer := webserver.New(webserver.Opts{
		Port:        config.Http.Port,
		Host:        config.Http.Host,
		DataStorage: ds,
		Logger:      loggerHttp,
	})

	err = ServiceStartup(ftpd.ListenAndServe, time.Millisecond*500)
	if err != nil {
		panic(fmt.Errorf("Can't start ftp server: %v", err))
	}

	err = ServiceStartup(webServer.Run, time.Millisecond*500)
	if err != nil {
		panic(fmt.Errorf("Can't start web server: %v", err))
	}

	uid := ug.New()
	_ = ds.Put(uid,
		&struct {
			Title   string
			Caption string
			Url     string
		}{"Title", "Caption", "https://starshiptroopers.dev"},
		nil,
	)

	logger.Printf("Data has been stored into the storage with uid: %s", uid)

	//waiting for the stop signal
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, os.Kill)
	select {
	case <-ch:
		_ = ftpd.Shutdown()
		webServer.Shutdown()
		fmt.Printf("\nThe server is shut down")
	}
}

func ServiceStartup(f func() error, waitTimeout time.Duration) error {
	closeCh := make(chan error)
	go func() {
		err := f()
		if err != nil {
			closeCh <- err
		}
		close(closeCh)
	}()

	select {
	case err := <-closeCh:
		return fmt.Errorf("Can't start a service: %v", err)
	case <-time.After(waitTimeout):

	}
	return nil
}
