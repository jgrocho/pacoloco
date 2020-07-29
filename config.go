package main

import (
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os/user"
	"strings"
)

const DefaultPort = 9129
const DefaultCacheDir = "/var/cache/pacoloco"

const (
	PurgeStrategyNone  = "none"
	PurgeStrategyTime  = "time"
	PurgeStrategyCount = "count"
)

type Repo struct {
	Url  string   `yaml:"url"`
	Urls []string `yaml:"urls"`
}

type Config struct {
	CacheDir        string          `yaml:"cache_dir"`
	Port            int             `yaml:"port"`
	Repos           map[string]Repo `yaml:"repos,omitempty"`
	PurgeFilesAfter int             `yaml:"purge_files_after"`
	PurgeStrategy   string          `yaml:"purge_strategy"`
	PurgeKeepAtMost int             `yaml:"purge_keep_at_most"`
}

var config *Config

func readConfig(filename string) *Config {
	var result = &Config{
		CacheDir:        DefaultCacheDir,
		Port:            DefaultPort,
		PurgeFilesAfter: 3600 * 24 * 30, // purge files if they are not accessed for 30 days
		PurgeStrategy:   PurgeStrategyTime,
		PurgeKeepAtMost: 3,
	}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, &result)
	if err != nil {
		log.Fatal(err)
	}

	// validate config
	for name, repo := range result.Repos {
		if repo.Url != "" && len(repo.Urls) > 0 {
			log.Fatalf("repo '%v' specifies both url and urls parameters, please use only one of them", name)
		}
		if repo.Url == "" && len(repo.Urls) == 0 {
			log.Fatalf("please specify url for repo '%v'", name)
		}
	}

	switch strings.ToLower(result.PurgeStrategy) {
	case PurgeStrategyNone:
		result.PurgeStrategy = PurgeStrategyNone

	case PurgeStrategyTime:
		if result.PurgeFilesAfter < 10*60 {
			log.Fatalf("purge_files_after period is too low (%v) please specify at least 10 minutes", result.PurgeFilesAfter)
		}
		result.PurgeStrategy = PurgeStrategyTime

	case PurgeStrategyCount:
		if result.PurgeKeepAtMost < 1 {
			log.Fatalf("purge_keep_at_most is too low (%v) please specify a positive integer", result.PurgeKeepAtMost)
		}
		result.PurgeStrategy = PurgeStrategyCount

	default:
		log.Fatalf("purge_strategy must be one of %q, %q, or %q", PurgeStrategyTime, PurgeStrategyCount, PurgeStrategyNone)
	}

	if unix.Access(result.CacheDir, unix.R_OK|unix.W_OK) != nil {
		u, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("directory %v does not exist or isn't writable for user %v", result.CacheDir, u.Username)
	}

	return result
}
