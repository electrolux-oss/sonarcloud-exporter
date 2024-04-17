FROM golang:alpine as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/sonarcloud-exporter ./cmd/sonarcloud-exporter

FROM alpine

ENV ORGANIZATION=""
ENV SC_TOKEN=""
ENV LISTEN_ADDRESS="8080"
ENV LISTEN_PATH="/metrics"
ENV METRICS_NAME="qualityGate"

COPY --from=builder /app/sonarcloud-exporter /usr/bin/


ENTRYPOINT /usr/bin/sonarcloud-exporter -organization $ORGANIZATION -scToken $SC_TOKEN -listenAddress $LISTEN_ADDRESS -listenPath $LISTEN_PATH -metricsName $METRICS_NAME
