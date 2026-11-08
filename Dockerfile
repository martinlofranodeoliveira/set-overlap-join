FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/set-overlap-join ./cmd/set-overlap-join
RUN go test -v -race ./...

FROM debian:bookworm-slim

RUN useradd --uid 1000 --create-home lab
USER 1000:1000
WORKDIR /home/lab

COPY --from=builder /bin/set-overlap-join /usr/local/bin/set-overlap-join

ENTRYPOINT ["/usr/local/bin/set-overlap-join"]
CMD ["demo", "--assert"]
