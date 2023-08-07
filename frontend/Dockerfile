FROM golang:1.20 as builder
COPY go.mod go.sum /build/
WORKDIR /build
RUN go mod download

COPY ./cmd ./cmd
COPY ./pkg ./pkg

RUN useradd scratchuser &&  cd /build/ && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /frontend cmd/frontend/main.go

FROM alpine:3.18
COPY --from=builder /frontend /frontend
COPY --from=builder /etc/passwd /etc/passwd
USER scratchuser
ENTRYPOINT [ "/frontend" ]
