package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/Xiol/whatbin"
	"github.com/Xiol/whatbin/pkg/config"
	"github.com/Xiol/whatbin/pkg/notifiers/pushover"
	"github.com/Xiol/whatbin/pkg/notifiers/stdout"
	"github.com/Xiol/whatbin/pkg/providers/corby"
	"github.com/Xiol/whatbin/pkg/providers/salford"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	if err := config.Init(); err != nil {
		log.WithError(err).Fatal("config load failed")
	}

	if *debug || viper.GetBool("enable_debugging") {
		log.SetLevel(log.DebugLevel)
	}

	var p whatbin.Provider
	switch viper.GetString("provider") {
	case "salford":
		p = salford.New(viper.GetInt("house_number"), viper.GetString("postcode"), httpClient)
	case "corby":
		p = corby.New(viper.GetString("uprn"))
	default:
		log.WithField("provider", viper.GetString("provider")).Fatal("unknown provider")
	}

	var n whatbin.Notifier
	switch viper.GetString("notifier") {
	case "stdout":
		n = stdout.New()
	case "pushover":
		n = pushover.New(viper.GetString("pushover_api_token"), viper.GetStringSlice("pushover_users"))
	default:
		log.WithField("notifier", viper.GetString("notifier")).Fatal("unknown notifier")
	}

	if err := whatbin.Handle(p, n); err != nil {
		log.Fatal(err.Error())
	}
}
