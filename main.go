package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/adshao/go-binance/v2"
)

const (
	defaultUpdateInterval      = "30s"
	defaultBaseURL             = "https://api.binance.us"
	defaultHTTPServerPort      = 8090
	defaultPrometheusNamespace = "binance"
	defaultTrackAllPrices      = false
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

func main() {
	// Flag values
	flag.StringVar(&rConf.apiKey, "apiKey", "", "Binance API Key")
	flag.StringVar(&rConf.secretKey, "secretKey", "", "Binance API secret Key")
	flag.StringVar(&rConf.apiBaseURL, "apiBaseUrl", defaultBaseURL, "Binance base API URL")
	flag.StringVar(&rConf.updateInterval, "updateInterval", defaultUpdateInterval, "Binance update interval")
	flag.BoolVar(&rConf.trackAllPrices, "trackAllPrices", defaultTrackAllPrices, "Intructs to track all prices vs only prices of coins in seen in balance")
	flag.UintVar(&rConf.httpServerPort, "httpServerPort", defaultHTTPServerPort, "HTTP server port")
	flag.Parse()

	// Update interval parse
	updIval, err := time.ParseDuration(rConf.updateInterval)
	if err != nil {
		fmt.Printf("Could not parse update interval duration, %v", err)
		return
	}
	rConf.updateIval = updIval

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
		fmt.Printf("Starting HTTP server at %s\n", rConf.httpServ.Addr)
		rConf.httpServ.ListenAndServe()

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
		fmt.Printf("Updating....\n")

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
		fmt.Printf("Failed to get account Balances: %v\n", err)
		return err
	}
	for _, bal := range acc.Balances {
		free, _ := strconv.ParseFloat(bal.Free, 64)
		locked, _ := strconv.ParseFloat(bal.Locked, 64)

		if free+locked != 0 {
			rConf.balances.WithLabelValues(bal.Asset, "free").Set(free)
			rConf.balances.WithLabelValues(bal.Asset, "locked").Set(locked)
			rConf.observedWalletCoins[bal.Asset+"USD"] = true
		}
	}
	return nil
}

func updatePrices() error {
	prices, err := rConf.client.NewListPricesService().Do(context.Background())
	if err != nil {
		fmt.Printf("Failed to get prices: %v\n", err)
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
