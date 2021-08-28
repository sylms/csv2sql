FROM golang:1.16 as builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

RUN go get -u github.com/gobuffalo/packr/v2/packr2

COPY . .

# https://github.com/gobuffalo/packr/tree/master/v2#building-a-binary
RUN packr2 && \
    CGO_ENABLED=0 go build -o csv2sql .

# runner
FROM alpine

COPY --from=builder /workspace/csv2sql /app/

RUN apk --no-cache add \
    tzdata \
    openssl

ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

CMD ["/app/csv2sql"]
