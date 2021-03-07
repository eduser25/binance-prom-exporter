package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	defaultTrackAllSymbols     = false
	defaultDebug               = false
	defaultLogLevel            = log.InfoLevel
	defaultPriceSymbol         = "USD"
)

type runtimeConfStruct struct {
	apiKey     string
	secretKey  string
	apiBaseURL string

	coinPrices *prometheus.GaugeVec
	balances   *prometheus.GaugeVec

	symbolsOFInterest map[string]string
	trackAllSymbols   bool
	manualSymbols     string
	pricesSymbol      string

	client         *binance.Client
	registry       *prometheus.Registry
	httpServerPort uint
	httpServ       *http.Server

	updateInterval string
	updateIval     time.Duration
	debug          bool
}

var rConf runtimeConfStruct = runtimeConfStruct{
	apiKey:     "",
	secretKey:  "",
	apiBaseURL: "",
	coinPrices: nil,
	balances:   nil,

	symbolsOFInterest: make(map[string]string),
	manualSymbols:     "",
	trackAllSymbols:   false,

	client:         nil,
	registry:       prometheus.NewRegistry(),
	httpServerPort: 0,
	httpServ:       nil,

	pricesSymbol:   "",
	updateInterval: "",
	updateIval:     0,
}

func parsePriceCoins(symbolList string) {
	syms := strings.Split(symbolList, ",")

	for _, coinSym := range syms {
		// This maps market symbols to the specific coin we are interested in
		log.Debugf("> Tracking %s", strings.Join([]string{coinSym, rConf.pricesSymbol}, ""))
		rConf.symbolsOFInterest[coinSym+rConf.pricesSymbol] = coinSym
	}
}

func initParams() {
	// Flag values
	flag.StringVar(&rConf.apiKey, "apiKey", "", "Binance API Key.")
	flag.StringVar(&rConf.secretKey, "apiSecret", "", "Binance API secret Key.")
	flag.StringVar(&rConf.apiBaseURL, "apiBaseUrl", defaultBaseURL, "Binance base API URL.")
	flag.StringVar(&rConf.updateInterval, "updateInterval", defaultUpdateInterval, "Binance update interval")
	flag.UintVar(&rConf.httpServerPort, "httpServerPort", defaultHTTPServerPort, "HTTP server port.")
	flag.BoolVar(&rConf.debug, "debug", defaultDebug, "Set debug log level.")
	flag.BoolVar(&rConf.trackAllSymbols, "trackAll", defaultTrackAllSymbols, "Will set to track all market symbols.")
	flag.StringVar(&rConf.pricesSymbol, "priceSymbol", defaultPriceSymbol, "Set the default baseline currency symbol to calculate prices.")

	manualPrices := ""
	flag.StringVar(&manualPrices, "symbols", "", "Manually set the curency symbols to track (uses baseline to get market symbols).")
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

	parsePriceCoins(manualPrices)
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

			log.Debugf("> Observing free in wallet %f for %s", free, bal.Asset)
			rConf.balances.WithLabelValues(bal.Asset, "free").Set(free)
			log.Debugf("> Observing locked in wallet %f for %s", locked, bal.Asset)
			rConf.balances.WithLabelValues(bal.Asset, "locked").Set(locked)

			if _, found := rConf.symbolsOFInterest[bal.Asset+rConf.pricesSymbol]; !found && (rConf.pricesSymbol != bal.Asset) {
				// This is a simple way to map trade market symbols (BTCUSD) to the asset we hold in wallet (BTC)
				log.Debugf("> Tracking %s", bal.Asset+rConf.pricesSymbol)
				rConf.symbolsOFInterest[bal.Asset+rConf.pricesSymbol] = bal.Asset
			}
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
		if !rConf.trackAllSymbols {
			if symbol, ok := rConf.symbolsOFInterest[p.Symbol]; ok {
				// This observes values with asset label value (ie. BTC) and obviates baseline currency in label,
				// This is made like so to later have prometheus be able to easilly match wallet symbols and price symbols
				// (ie. Balance calculation vector `prices * wallet` will automatically match the right labels)
				price, _ := strconv.ParseFloat(p.Price, 64)
				log.Debugf("> Observing %f for %s", price, symbol)
				rConf.coinPrices.WithLabelValues(symbol).Set(price)
			}
		} else {
			// This observes values with full market symbol label value (BTCETH)
			price, _ := strconv.ParseFloat(p.Price, 64)
			log.Debugf("> Observing %f for %s", price, p.Symbol)
			rConf.coinPrices.WithLabelValues(p.Symbol).Set(price)
		}
	}
	return nil
}
