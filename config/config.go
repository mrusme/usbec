package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Cmd struct {
	Command string
	Args    []string
}

type Device struct {
	ID         string
	VendorID   int
	ProductID  int
	PrettyName string
	On         struct {
		Attach []Cmd
		Detach []Cmd
	}
	NotificationIcon string
}

type Config struct {
	Debug         bool
	Notifications bool
	Devices       []Device
}

func Cfg() (Config, error) {
	viper.SetDefault("Debug", "false")
	viper.SetDefault("Notifications", "false")

	viper.SetConfigName("usbec.toml")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath("$XDG_CONFIG_HOME/")
	viper.AddConfigPath("$HOME/.config/")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("USBEC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}
