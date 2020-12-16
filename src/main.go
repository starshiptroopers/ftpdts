package main

import (
	"fmt"
	"ftpdts/src/storage"
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

	ug := uidgenerator.New(
		&uidgenerator.Cfg{
			Alfa:      "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
			Format:    "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
			Validator: "[0-9a-zA-Z]{32}",
		},
	)

	ts := tmplstorage.New(config.Templates.Path)

	datastorage.DefaultCacheTTL = time.Second * time.Duration(config.Cache.DataTTL)
	memoryDs := datastorage.NewMemoryDataStorage()
	fsDs := storage.NewFsDataStorage(config.Data.Path, ug)

	var forever = time.Duration(0)
	var cnt = 0
	err = fsDs.Pass(func(uid string, createdAt time.Time, data interface{}) {
		if err := memoryDs.Put(uid, data, &forever); err != nil {
			panic(fmt.Errorf("something wrong with loading persistent data into the memory cache: %v", err))
		}
		cnt++
	})
	if err != nil {
		panic(fmt.Errorf("can't initialize the data persistent storage: %v", err))
	}
	logger.Printf("%d persistent data records has been loaded into the data memory cache", cnt)

	ftpOpts := &core.ServerOpts{
		Port:     int(config.Ftp.Port),
		Hostname: config.Ftp.Host,
	}

	ftpd := ftpdt.New(
		&ftpdt.Opts{
			FtpOpts:         ftpOpts,
			TemplateStorage: ts,
			DataStorage:     memoryDs,
			UidGenerator:    ug,
			LogWriter:       loggerFtp.Writer(),
			LogFtpDebug:     false,
		},
	)

	webServer := webserver.New(webserver.Opts{
		Port:           config.Http.Port,
		Host:           config.Http.Host,
		DataStorage:    storage.NewDataStorage(memoryDs, fsDs),
		Logger:         loggerHttp,
		UidGenerator:   ug,
		MaxRequestBody: config.Http.MaxRequestBody,
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
	_ = memoryDs.Put(uid,
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
