FROM golang:1.20 as builder
COPY go.mod go.sum /build/
WORKDIR /build
RUN go mod download

COPY ./main.go ./main.go
RUN useradd scratchuser &&  cd /build/ && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /server .

FROM alpine:3.18
COPY --from=builder /server /server
COPY --from=builder /etc/passwd /etc/passwd
USER scratchuser
CMD [ "/server" ]
