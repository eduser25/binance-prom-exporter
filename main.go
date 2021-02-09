package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	defaultUpdateInterval      = "30s"
	defaultBaseURL             = "https://api.binance.us"
	defaultHTTPServerPort      = 8090
	defaultPrometheusNamespace = "binance"
	defaultTrackAllPrices      = false
	defaultDebug               = false
	defaultLogLevel            = log.InfoLevel
	defaultPriceSymbol         = "USD"
)

type runtimeConfStruct struct {
	apiKey     string
	secretKey  string
	apiBaseURL string

	trackAllPrices bool

	coinPrices          *prometheus.GaugeVec
	balances            *prometheus.GaugeVec
	observedWalletCoins map[string]bool

	client         *binance.Client
	registry       *prometheus.Registry
	httpServerPort uint
	httpServ       *http.Server

	updateInterval string
	updateIval     time.Duration
	debug          bool
}

var rConf runtimeConfStruct = runtimeConfStruct{
	apiKey:              "",
	secretKey:           "",
	apiBaseURL:          "",
	trackAllPrices:      false,
	coinPrices:          nil,
	balances:            nil,
	observedWalletCoins: make(map[string]bool),
	client:              nil,
	registry:            prometheus.NewRegistry(),
	httpServerPort:      0,
	httpServ:            nil,

	updateInterval: "",
	updateIval:     0,
}

func initParams() {
	// Flag values
	flag.StringVar(&rConf.apiKey, "apiKey", "", "Binance API Key")
	flag.StringVar(&rConf.secretKey, "apiSecret", "", "Binance API secret Key")
	flag.StringVar(&rConf.apiBaseURL, "apiBaseUrl", defaultBaseURL, "Binance base API URL")
	flag.StringVar(&rConf.updateInterval, "updateInterval", defaultUpdateInterval, "Binance update interval")
	flag.BoolVar(&rConf.trackAllPrices, "trackAllPrices", defaultTrackAllPrices, "Intructs to track all prices vs only prices of coins in seen in balance")
	flag.UintVar(&rConf.httpServerPort, "httpServerPort", defaultHTTPServerPort, "HTTP server port")
	flag.BoolVar(&rConf.debug, "debug", defaultDebug, "Set debug log level")
	flag.Parse()

	logLvl := defaultLogLevel
	if rConf.debug {
		logLvl = log.DebugLevel
	}
	log.SetLevel(logLvl)

	// Update interval parse
	updIval, err := time.ParseDuration(rConf.updateInterval)
	if err != nil {
		log.Errorf("Could not parse update interval duration, %v", err)
		os.Exit(-1)
	}
	rConf.updateIval = updIval
}

func main() {
	// basic init and parsing of flags
	initParams()

	// Init binance client
	rConf.client = binance.NewClient(rConf.apiKey, rConf.secretKey)
	rConf.client.BaseURL = rConf.apiBaseURL

	// Register prom metrics path in http serv
	httpMux := http.NewServeMux()
	httpMux.Handle("/metrics", promhttp.InstrumentMetricHandler(
		rConf.registry,
		promhttp.HandlerFor(rConf.registry, promhttp.HandlerOpts{}),
	))

	// Init & start serv
	rConf.httpServ = &http.Server{
		Addr:    fmt.Sprintf(":%d", rConf.httpServerPort),
		Handler: httpMux,
	}
	go func() {
		log.Infof("> Starting HTTP server at %s\n", rConf.httpServ.Addr)
		err := rConf.httpServ.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Errorf("HTTP Server errored out %v", err)
		}
	}()

	// Init Prometheus Gauge Vectors
	rConf.balances = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultPrometheusNamespace,
		Name:      "balance",
		Help:      fmt.Sprintf("Balance in account for assets"),
	},
		[]string{"symbol", "status"},
	)
	rConf.registry.MustRegister(rConf.balances)

	rConf.coinPrices = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: defaultPrometheusNamespace,
		Name:      "price",
		Help:      fmt.Sprintf("Symbol prices"),
	},
		[]string{"symbol"},
	)
	rConf.registry.MustRegister(rConf.coinPrices)

	// Regular loop operations below
	ticker := time.NewTicker(rConf.updateIval)
	for {
		log.Debug("> Updating....\n")

		// Update balances
		updateAccountBalances()

		// Update prices
		updatePrices()

		<-ticker.C
	}
}

func updateAccountBalances() error {
	acc, err := rConf.client.NewGetAccountService().Do(context.Background())
	if err != nil {
		log.Errorf("Failed to get account Balances: %v\n", err)
		return err
	}
	for _, bal := range acc.Balances {
		free, _ := strconv.ParseFloat(bal.Free, 64)
		locked, _ := strconv.ParseFloat(bal.Locked, 64)

		if free+locked != 0 {
			rConf.balances.WithLabelValues(bal.Asset, "free").Set(free)
			rConf.balances.WithLabelValues(bal.Asset, "locked").Set(locked)
			// This is an easy fix to later have the market trade symbols
			// lookup successfully on the coin symbols we have
			rConf.observedWalletCoins[bal.Asset+defaultPriceSymbol] = true
		}
	}
	return nil
}

func updatePrices() error {
	prices, err := rConf.client.NewListPricesService().Do(context.Background())
	if err != nil {
		log.Errorf("Failed to get prices: %v\n", err)
		return err
	}

	for _, p := range prices {
		if _, ok := rConf.observedWalletCoins[p.Symbol]; ok || rConf.trackAllPrices {
			price, _ := strconv.ParseFloat(p.Price, 64)
			rConf.coinPrices.WithLabelValues(p.Symbol).Set(price)
		}
	}
	return nil
}
