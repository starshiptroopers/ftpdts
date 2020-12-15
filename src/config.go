package main

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/spf13/viper"
	"reflect"
)

var (
	defConfigPath      = "."
	defConfigName      = "ftpdts.ini"
	defConfigType      = "ini"
	defConfigEnvPrefix = "FTPDTS"
)

type Config struct {
	Http struct {
		Port           uint   `default:"2001"`
		Host           string `default:"127.0.0.1"`
		MaxRequestBody int64  `default:"1024"`
	}

	Ftp struct {
		Port uint   `default:"2000"`
		Host string `default:"127.0.0.1"`
	}

	Templates struct {
		Path string `default:"./tmpl"`
	}

	Logs struct {
		Ftp             string `default:"logs/ftp.log"`
		FtpNoConsole    bool   `default:"false"`
		Http            string `default:"logs/http.log"`
		HttpNoConsole   bool   `default:"false"`
		Ftpdts          string `default:"logs/ftpdts.log"`
		FtpdtsNoConsole bool   `default:"false"`
	}

	Cache struct {
		DataTTL uint `default:"86400"`
	}
}

func viperConfig(cPath string, cName string, cType string, envPrefix string, config interface{}) (v *viper.Viper, err error) {
	v = viper.New()
	v.AddConfigPath(cPath)
	v.SetConfigType(cType)
	v.AutomaticEnv()
	v.SetEnvPrefix(envPrefix)
	v.SetConfigName(cName)
	//fill default config values
	if reflect.TypeOf(config).Kind() == reflect.Ptr && reflect.ValueOf(config).Elem().Type().Kind() == reflect.Struct {
		if err = defaults.Set(config); err != nil {
			return
		}
	}
	return
}

func LoadConfig() (c *Config, err error) {

	c = new(Config)
	viperConfig, err := viperConfig(defConfigPath, defConfigName, defConfigType, defConfigEnvPrefix, c)
	if err != nil {
		return
	}

	// Find and read the config file
	if err := viperConfig.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("wrong config file: %v", err)
	}

	if err := viperConfig.Unmarshal(c); err != nil {
		return nil, fmt.Errorf("unable to parse config file: %v", err)
	}

	return
}
