############################################
# 3 stage build
#
# build container first
############################################
FROM golang:alpine AS build-go

# Copy the local package files to the container's workspace.
ADD . /src

# build Go executable
WORKDIR /src/example

# not ideal, but it's the best I could do on short notice
RUN GOOS=linux go build -mod=vendor -a -tags netgo -ldflags '-w' .

############################################
# Then run tests in another container
############################################
FROM build-go
RUN apk add --no-cache libc-dev gcc redis bash
WORKDIR /src/example/redis
RUN (./redis-run.sh > redis.out & ) && (sleep 5) && (./sentinel-run.sh > sentinel.out &) && cd /src &&  go test -mod=vendor


############################################
# deployment container 
############################################
FROM alpine
RUN apk --no-cache add ca-certificates redis bash
COPY --from=build-go /src/example/example /example
COPY --from=build-go /src/docker-run-example-with-deps.sh /docker-run-example-with-deps.sh
COPY --from=build-go /src/example/redis/redis-run.sh /redis-run.sh
COPY --from=build-go /src/example/redis/sentinel-run.sh /sentinel-run.sh
COPY --from=build-go /src/example/redis/conf/* /conf/
RUN mkdir -p /run

ENV        PORT 9090
EXPOSE     9090
ENTRYPOINT ["./docker-run-example-with-deps.sh"]