# binance-prom-exporter
Simple binance wallet and coin price exporter to prometheus format.

```
./binance-prom-exporter -apiKey=<apiKey> -apiSecret=<secretKey> -priceSymbol "USD" --symbols "DOGE,BTC"
INFO[0000] > Starting HTTP server at :8090
...
```
You can later explore how insanely rich you like:
```
$ curl localhost:8090/metrics
# HELP binance_balance Balance in account for assets
# TYPE binance_balance gauge
binance_balance{status="free",symbol="ATOM"} 1425.79956193
binance_balance{status="free",symbol="ETH"} 9867.36777585
binance_balance{status="free",symbol="BTC"} 1.567890
binance_balance{status="free",symbol="LINK"} 1115.52042
binance_balance{status="free",symbol="USD"} 711241.5632
binance_balance{status="free",symbol="DOGE"} 1000000.00000
binance_balance{status="locked",symbol="ATOM"} 0
binance_balance{status="locked",symbol="ETH"} 0
binance_balance{status="locked",symbol="LINK"} 0
binance_balance{status="locked",symbol="USD"} 0
# HELP binance_price Symbol prices
# TYPE binance_price gauge
binance_price{symbol="ATOM"} 18.928
binance_price{symbol="BTC"} 50504.46
binance_price{symbol="DOGE"} 0.0513
binance_price{symbol="ETH"} 1653.97
binance_price{symbol="LINK"} 27.7139
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 1
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
```

By default, only symbols seen on wallet will be tracked/exported using currency baseline for market.
- `priceSymbol` is the baseline where all coins will be compared against. `USD` by default.
- `symbols` allow adding additional symbols of interest that are not observed in wallet (see above). Ie. if you don't
own BTC but still interested in tracking BTCUSD price trend.

Labels for wallet and market prices will be the currency symbol in the previous case, obviating the currency baseline.
This makes it easy to calc later balances through PromQL as labels will match with no further label juggling.

- `trackAll` allows tracking all market symbols.

Market labels here are in full.

Supports binance.com, uses binance.us by default though.

```
Usage of ./binance-prom-exporter:
  -apiBaseUrl string
    	Binance base API URL. (default "https://api.binance.us")
  -apiKey string
    	Binance API Key.
  -apiSecret string
    	Binance API secret Key.
  -debug
    	Set debug log level.
  -httpServerPort uint
    	HTTP server port. (default 8090)
  -priceSymbol string
    	Set the default baseline currency symbol to calculate prices. (default "USD")
  -symbols string
    	Manually set the curency symbols to track (uses baseline to get market symbols).
  -trackAll
    	Will set to track all market symbols.
  -updateInterval string
    	Binance update interval (default "30s")
 ```
 
 
