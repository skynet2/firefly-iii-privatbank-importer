FROM golang:1.22-alpine as build
ADD . /src
WORKDIR /src/cmd/balances
RUN GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /bin/balances

FROM alpine:latest
COPY --from=build /bin/balances /opt/app/balances
ADD ./images/balances/entrypoint.sh /opt/app/entrypoint.sh
RUN chmod +x /opt/app/entrypoint.sh
RUN chmod +x /opt/app/balances
WORKDIR /opt/app
ENTRYPOINT ["./entrypoint.sh"]