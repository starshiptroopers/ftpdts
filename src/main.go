// Copyright 2020 The Starship Troopers Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// ftpdts Ftp server based on ftpdt and goftp/server and used to do a real-time files generation from templates and exposes them as downloadable files at the built-in ftp server.
// It was originally designed to implement the trick of escaping to the default browser from Facebook, Instagram, webkit in-app browsers on IOS.
// Ftp is used as an intermediate gateway to download the html file with redirect to real web site.
// The main reason the ftp is using in this trick is because the ftp protocol is handled with Safary by default.
// Opening a ftp link in a webkit lead to starting the Safari. At the moment of developing this library it was the only way to escape from Facebook browser to ios default browser.
//
// Configuration:
// Server read the configuration from in ftpdts.ini file
// Ftp server is listening at 2001 port, http server (rest api endpoints) is listening at 2000 default port
// Templates is stored at ./tmpl folder by default
// Persistent data storage is at ./data folder
//
// Templates:
// default.tmpl is the default template file. It is used when the ftp client requests the file from the root folder, for example with url: ftp://server/UID.html
// You can customize your templates and place them into templates folder with a different filename. The template file name uses as ftp server folder name.
// For example, we place a template with name test.tmpl into the templates folder, then we can use the url ftp://server/test/UID.html to download the file created from this template and populated with UID data set
//
// Datasets:
// You can create your default dataset and place them into the ./data folder as file where name is in UID format. The file must contain a JSON
// or you can post dataset directly to the rest api endpoint and it will be stored into the memory cache and persistent storage

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

var gitTag, gitCommit, gitBranch string

func main() {

	if gitTag != "" {
		fmt.Printf("Ftpdts service version %s (%s, %s)", gitTag, gitBranch, gitCommit)
	}

	config, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	loggerFTP := logInit(config.Logs.FTP, !config.Logs.FTPNoConsole)
	loggerHTTP := logInit(config.Logs.HTTP, !config.Logs.HTTPNoConsole)
	logger := logInit(config.Logs.Ftpdts, !config.Logs.FtpdtsNoConsole)

	ug := uidgenerator.New(
		&uidgenerator.Cfg{
			Alfa:      config.UID.Chars,
			Format:    config.UID.Format,
			Validator: config.UID.ValidatorRegexp,
		},
	)

	ts := tmplstorage.New(config.Templates.Path)

	datastorage.DefaultCacheTTL = time.Second * time.Duration(config.Cache.DataTTL)
	memoryDs := datastorage.NewMemoryDataStorage()
	fsDs := storage.NewFsDataStorage(config.Data.Path, ug)

	var forever = time.Duration(0)

	//Load data from the persistent storage
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
		Port:     int(config.FTP.Port),
		Hostname: config.FTP.Host,
	}

	ftpd := ftpdt.New(
		&ftpdt.Opts{
			FtpOpts:         ftpOpts,
			TemplateStorage: ts,
			DataStorage:     memoryDs,
			UidGenerator:    ug,
			LogWriter:       loggerFTP.Writer(),
			LogFtpDebug:     false,
		},
	)

	webServer := webserver.New(webserver.Opts{
		Port:           config.HTTP.Port,
		Host:           config.HTTP.Host,
		DataStorage:    storage.NewDataStorage(memoryDs, fsDs),
		Logger:         loggerHTTP,
		UIDGenerator:   ug,
		MaxRequestBody: config.HTTP.MaxRequestBody,
	})

	err = ServiceStartup(ftpd.ListenAndServe, time.Millisecond*500)
	if err != nil {
		panic(fmt.Errorf("can't start ftp server: %v", err))
	}

	err = ServiceStartup(webServer.Run, time.Millisecond*500)
	if err != nil {
		panic(fmt.Errorf("can't start web server: %v", err))
	}

	//test data
	/*
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
		/*

	*/
	//waiting for the stop signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	<-ch
	_ = ftpd.Shutdown()
	webServer.Shutdown()
	fmt.Printf("\nThe server is shut down")

}

//starts the service (f func) as a gorutine and wait waitTimeout to service became ready
//returns err if service returns err in waitTimeout time
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
