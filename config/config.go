package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port    string
	Volumes []string
}

func New(v *viper.Viper) (*Config, error) {
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	//if v.IsSet("config") {
	if v.GetString("config") != "" {
		v.SetConfigFile(v.GetString("config"))
		err := v.ReadInConfig()
		if err != nil {
			return nil, err
		}

	} /* else {*/
	//pwd, err := os.Getwd()
	//if err != nil {
	//return nil, err
	//}

	//v.SetConfigName("rebost")
	//v.AddConfigPath(pwd)
	/*}*/
	//err := v.ReadInConfig()
	//if err != nil {
	//return nil, err
	//}

	return &Config{
		Port:    v.GetString("port"),
		Volumes: v.GetStringSlice("volumes"),
	}, nil
}
