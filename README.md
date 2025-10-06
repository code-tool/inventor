# inventor

Prometheus HTTP SD implementation

## Description

The Inventor is a Prometheus HTTP SD Server allows users to dynamcially add or remove prometheus targets and labels and expose it to a [Prometheus HTTP SD](https://prometheus.io/docs/prometheus/latest/http_sd/) job.

## Usage

Running the server:
```
cd src
go run main.go
```

Installing with Helm
```bash
helm repo add inventor https://code-tool.github.io/inventor/
```

Registering new target:
```bash
curl -X PUT -H "x-api-token: secret" http://127.0.0.1:9101/target \
-d '{"static_config": {"targets": ["10.0.10.2:9100",], "labels": {"__meta_datacenter": "dc-01", "__meta_prometheus_job": "node"}, "target_group": "mygroup"}}'
```

More examples: `./test/end-to-end`

Prometheus SD config example
```yaml
scrape_configs:
  - job_name: http_sd
    http_sd_configs:
      - url: http://127.0.0.1:9101/discover
        # if SD_TOKEN env variable is set
        headers:
          - "x-sd-token: REDACTED"

```

Prometheus SD config with groups example
```yaml
scrape_configs:
  - job_name: http_sd_mygroup
    http_sd_configs:
      - url: http://127.0.0.1:9101/group?name=mygroup
        # if SD_TOKEN env variable is set
        headers:
          - "x-sd-token: REDACTED"

```


## Configuration Environmet Valiables

  * `REDIS_ADDR`: redis server addres to store metrics and targets
  * `REDIS_PORT`: redis server port
  * `REDIS_DBNO`: redis server keyspace
  * `TTL_SECONDS`: ttl for storing target, default is 6h (21600 seconds)
  * `API_TOKEN`: API token for manipulating targets
  * `SD_TOKEN`: Options token for Prometheus HTTP SD, is empty by default and not validating (header `x-sd-token`)


## Custom discovered labels

  * `__meta_inventor_sd_module`: contains element of `modules: []`, useful fo relabeling to add `__param_module`


## API Methods

* **GET /discover**
    * Returning the list of targets in Prometheus HTTP SD format
* **GET /group**
    * Returning targets by group name `/group?name=mygroup`
* **PUT /target**
    * Adds the new target
* **GET /target**
    * Returning target by ID
* **DELETE /target**
    * Removing target by ID
* **GET /metrics**
    * Metrics in prometheus format
* **GET /healthcheck**
    * Health Check for kubernetes deployments


## Build Docker image
```bash
docker build -t ghcr.io/code-tool/inventor/inventor:$(cat VERSION.txt) --build-arg BUILD_VERSION=$(cat VERSION.txt) -f docker/Dockerfile .
```
pulling image:
```bash
ghcr.io/code-tool/inventor/inventor:0.0.3
```

## License

Covered under the [MIT license](LICENSE.md).
