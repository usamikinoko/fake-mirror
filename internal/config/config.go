package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type SiteConfig struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
	Favicon     string `yaml:"favicon"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type HomeConfig struct {
	Title    string `yaml:"title"`
	SubTitle string `yaml:"subtitle"`
	Avatar   string `yaml:"avatar"`
	Owner    string `yaml:"owner"`
}

type DeployServerConfig struct {
	Host       string `yaml:"host"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Port       int    `yaml:"port"`
	Path       string `yaml:"path"`
	Identity   string `yaml:"identity"`
	KnownHosts string `yaml:"known_hosts"`
}

type DeepConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Password string `yaml:"password"`
	Sign     string `yaml:"sign"`
}

type DeployConfig struct {
	Mode   string             `yaml:"mode"`
	Remote string             `yaml:"remote"`
	Branch string             `yaml:"branch"`
	Server DeployServerConfig `yaml:"server"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Site   SiteConfig   `yaml:"site"`
	Home   HomeConfig   `yaml:"home"`
	Deep   DeepConfig   `yaml:"deep"`
	Deploy DeployConfig `yaml:"deploy"`
}

var Cfg *Config

func Load() error {
	data, err := os.ReadFile("_config.yaml")
	if err != nil {
		return err
	}

	Cfg = &Config{}
	if err := yaml.Unmarshal(data, Cfg); err != nil {
		return err
	}

	Cfg.Site.URL = strings.TrimSpace(Cfg.Site.URL)
	Cfg.Site.Description = strings.TrimSpace(Cfg.Site.Description)
	Cfg.Site.Favicon = strings.TrimSpace(Cfg.Site.Favicon)
	Cfg.Home.Title = strings.TrimSpace(Cfg.Home.Title)
	Cfg.Home.SubTitle = strings.TrimSpace(Cfg.Home.SubTitle)
	Cfg.Home.Avatar = strings.TrimSpace(Cfg.Home.Avatar)
	Cfg.Home.Owner = strings.TrimSpace(Cfg.Home.Owner)
	Cfg.Deep.Password = strings.TrimSpace(Cfg.Deep.Password)
	Cfg.Deep.Sign = strings.TrimSpace(Cfg.Deep.Sign)
	Cfg.Deploy.Mode = strings.TrimSpace(Cfg.Deploy.Mode)
	Cfg.Deploy.Remote = strings.TrimSpace(Cfg.Deploy.Remote)
	Cfg.Deploy.Branch = strings.TrimSpace(Cfg.Deploy.Branch)
	Cfg.Deploy.Server.Host = strings.TrimSpace(Cfg.Deploy.Server.Host)
	Cfg.Deploy.Server.User = strings.TrimSpace(Cfg.Deploy.Server.User)
	Cfg.Deploy.Server.Password = strings.TrimSpace(Cfg.Deploy.Server.Password)
	Cfg.Deploy.Server.Path = strings.TrimSpace(Cfg.Deploy.Server.Path)
	Cfg.Deploy.Server.Identity = strings.TrimSpace(Cfg.Deploy.Server.Identity)
	Cfg.Deploy.Server.KnownHosts = strings.TrimSpace(Cfg.Deploy.Server.KnownHosts)

	if Cfg.Server.Port == 0 {
		Cfg.Server.Port = 8080
	}
	if Cfg.Deploy.Mode == "" {
		Cfg.Deploy.Mode = "git"
	} else if Cfg.Deploy.Mode != "git" && Cfg.Deploy.Mode != "server" {
		return fmt.Errorf("unsupported deploy.mode %q", Cfg.Deploy.Mode)
	}
	if Cfg.Deploy.Branch == "" {
		Cfg.Deploy.Branch = "gh-pages"
	}
	if Cfg.Deploy.Server.Port == 0 {
		Cfg.Deploy.Server.Port = 22
	}
	if Cfg.Deploy.Server.Path == "" {
		Cfg.Deploy.Server.Path = "/var/www/rainhush"
	}

	return nil
}
