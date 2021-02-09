# binance-pricer
Simple binance wallet and coin price exporter to prometheus format.

```
./binance-pricer -apiKey=<apiKey> -apiSecret=<secretKey>
INFO[0000] > Starting HTTP server at :8090
...
```
You can later explore how insanely rich you like:
```
$ curl localhost:8090/metrics
# HELP binance_balance Balance in account for assets
# TYPE binance_balance gauge
binance_balance{status="free",symbol="ATOM"} 245.982
binance_balance{status="free",symbol="ETH"} 924851.245 # I wish
binance_balance{status="free",symbol="LINK"} 59.52042
binance_balance{status="free",symbol="USD"} 71.5632
binance_balance{status="locked",symbol="ATOM"} 0
binance_balance{status="locked",symbol="ETH"} 0
binance_balance{status="locked",symbol="LINK"} 0
binance_balance{status="locked",symbol="USD"} 0
# HELP binance_price Symbol prices
# TYPE binance_price gauge
binance_price{symbol="ATOMUSD"} 14.93
binance_price{symbol="ETHUSD"} 1740.47
binance_price{symbol="LINKUSD"} 25.5461
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 1
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
```

Supports binance.com, uses binance.us by default though.
By default, exports only coin values for coins in wallet where balance `!=0` (uses USD trade values) - can be overriden by `trackAllPrices`.

```
./binance-pricer  --help
Usage of ./binance-pricer:
  -apiBaseUrl string
    	Binance base API URL (default "https://api.binance.us")
  -apiKey string
    	Binance API Key
  -debug
    	Set debug log level
  -httpServerPort uint
    	HTTP server port (default 8090)
  -secretKey string
    	Binance API secret Key
  -trackAllPrices
    	Intructs to track all prices vs only prices of coins in seen in balance
  -updateInterval string
    	Binance update interval (default "30s")
 ```
 
 
