
ARG APP_VERSION
ARG GIT_COMMIT
ARG BUILD_DATE

FROM golang:1.24.5-alpine3.22 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . ./
RUN CGO_ENABLED=0 go build \
    -ldflags \
        "-X main.appVersion=${APP_VERSION} \
        -X main.gitCommit=${GIT_COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \ 
    -v \
    -o /usr/local/bin/httpbin ./cmd/server

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/trevatk/httpbin" 

COPY --from=builder /usr/local/bin/httpbin ./

CMD [ "./httpbin" ]
