FROM golang:1.16-alpine as build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY *.go .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /binary

FROM scratch

COPY --from=build /binary /binary

EXPOSE 8080

CMD [ "/binary" ]
