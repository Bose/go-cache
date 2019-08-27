# Run example 
```bash
go build && ./example
```

```bash
curl http://localhost:9090/cached-page

curl http://localhost:9090/cached-encrypted-entry

curl http://localhost:9090/cached-in-memory-not-encrypted-entry

curl localhost:9090/cached-in-memory-lru-with-expiry

# see metrics
curl localhost:9090/metrics

```

## Install/run redis and sentinel

```bash
# build and install redis-4.0.10.tar.gz
redis-run.sh
sentinel-run.sh
```

