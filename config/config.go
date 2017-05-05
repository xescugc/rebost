package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Config struct {
	Volumes []string `json:"volumes"`
}

type stringslice []string

func (s *stringslice) String() string {
	return fmt.Sprint(*s)
}

func (s *stringslice) Set(v string) error {
	*s = append(*s, strings.Split(v, ",")...)
	return nil
}

func New() *Config {

	var (
		volumes stringslice
		config  string
	)

	flag.StringVar(&config, "config", "", "Config full path")
	flag.StringVar(&config, "c", "", "Config full path (shorthand)")

	flag.Var(&volumes, "volume", "List of volumes (multi value or ',' separated)")
	flag.Var(&volumes, "v", "List of volumes, multi value or ',' separated (shorthand)")

	flag.Parse()

	c := loadConfig(config)

	if len(volumes) != 0 {
		c.Volumes = volumes
	}

	return c
}

func loadConfig(config string) *Config {
	var p string
	if config == "" {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		p = path.Join(wd, "rebost.json")
	} else {
		p = config
	}

	data, err := ioutil.ReadFile(p)
	if err != nil && os.IsNotExist(err) && config != "" {
		panic(err)
	}

	var c Config
	err = json.Unmarshal(data, &c)
	if err != nil {
		panic(err)
	}
	return &c
}
