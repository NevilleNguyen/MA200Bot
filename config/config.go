package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	ConfigFile string
)

func InitConfig() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	l := zap.S()

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	if ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(ConfigFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath("./env/")
		viper.AddConfigPath("../env/")
		viper.AddConfigPath("../../env/")
		viper.AddConfigPath(basepath)
		viper.SetConfigType("json")

		network := os.Getenv("NETWORK")
		switch network {
		case "mainnet":
			viper.SetConfigName("mainnet")
		case "ropsten":
			viper.SetConfigName("ropsten")
		case "testnet":
			viper.SetConfigFile("testnet")
		default:
			viper.SetConfigName("mainnet")
		}

	}

	envFilepath := filepath.Join(basepath, "../.env")
	if err := godotenv.Load(envFilepath); err != nil {
		l.Panicf("error load .env file")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		l.Infof("Using config file: %v", viper.ConfigFileUsed())
	}
}
