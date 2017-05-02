# telegraf_plugin_kafka_consumer_ssl

## Why this plugin

When I trying to use telegraf as the consumer of kafka(it means an input for telegraf), I found that the input plugin kafka_consumer couldn't fill my needs, and neither do the library it uses, so I made this one, using this library [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go).
One important thing you need to know is that [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go) is a wrapper around [librdkafka](https://github.com/edenhill/librdkafka), which is a C/C++ library, so you need to have this library before using this plugin.

## What could it do

This plugin allow you easliy use SSL connection between kafka and the telegraf input end.
It works with latest telegraf source code. (2017-05-02)

## How to use and config this plugin

* I assume you already have all keys and certs, if you don't, please follow this page [Using-SSL-with-librdkafka](https://github.com/edenhill/librdkafka/wiki/Using-SSL-with-librdkafka)
* Check the property ```ssl.client.auth``` in kafka server end.
* If the value of ```ssl.client.auth``` is ```required```, than the config part for this plugin in telegraf.conf should be like this:
``` 
[[inputs.kafka_consumer]]
    brokers = "127.0.0.1:9093"
    topics = ["cpu"]
    consumer_group = "telegraf_metrics_consumers"
    data_format = "influx"
    ssl_enabled = true
    ssl_ca = "/etc/ssl/ca-cert"
    ssl_cert = "/etc/ssl/kafka-client.pem"
    ssl_key = "/etc/ssl/kafka-client.key"
    #pwd could left empty if the key file don't has one.
    ssl_keypwd = "XXXXXXX"
```
* If the value of ```ssl.client.auth``` is ```none```, than the config part for this plugin in telegraf.conf should be like this:
``` 
[[inputs.kafka_consumer]]
    brokers = "127.0.0.1:9093"
    topics = ["cpu"]
    consumer_group = "telegraf_metrics_consumers"
    data_format = "influx"
    ssl_enabled = true
    ssl_ca = "/etc/ssl/ca-cert"
```


## How to compile

* First, get [telegraf](https://github.com/influxdata/telegraf)
* Then, pls compile it with Golang.
* Use this kafka_consumer.go to replace the one in ```plugins/inputs/kafka_consumer/kafka_consumer.go```
* Make a new telegraf binary and enjoy it!




## Appendix: Enable SSL in kafka server end

* SSL config in kafka server end, add the following config in kafka's ```server.properties``` could start a normal service in port 9092 and a SSL service in 9093

```
host.name=127.0.0.1
advertised.host.name=127.0.0.1
listeners=PLAINTEXT://127.0.0.1:9092,SSL://127.0.0.1:9093
advertised.listeners=PLAINTEXT://127.0.0.1:9092,SSL://127.0.0.1:9093

ssl.protocol=TLS
ssl.enabled.protocols=TLSv1.2,TLSv1.1,TLSv1
ssl.keystore.type=JKS
ssl.keystore.location=/path-to-file/kafka-server.keystore.jks
ssl.keystore.password=XXXXXX
ssl.key.password=XXXXXX
ssl.truststore.type=JKS
ssl.truststore.location=/path-to-file/kafka-server.truststore.jks
ssl.truststore.password=XXXXXX
# To require authentication of clients use "required", else "none" or "requested"
ssl.client.auth=required
```

Any problem, let me know: ```xiaoduo.lian@gmail.com```