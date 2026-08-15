# Dort Go servisi icin ortak imaj: docker build --build-arg SERVICE=gateway .
FROM golang:1.25-alpine AS build
WORKDIR /src

# Bagimliliklar kaynak koddan once indirilir ki kod degisince katman yeniden kullanilsin.
COPY go.work go.work.sum ./
COPY shared/go.mod ./shared/
COPY engine/go.mod engine/go.sum ./engine/
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY processor/go.mod processor/go.sum ./processor/
COPY marketmaker/go.mod ./marketmaker/
RUN go mod download

COPY shared ./shared
COPY engine ./engine
COPY gateway ./gateway
COPY processor ./processor
COPY marketmaker ./marketmaker

ARG SERVICE
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./${SERVICE}

FROM alpine:3.21
RUN adduser -D -u 10001 app && mkdir /data && chown app /data
USER app
COPY --from=build /out/app /usr/local/bin/app
ENTRYPOINT ["/usr/local/bin/app"]
