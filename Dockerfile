FROM golang:1.24 AS build

WORKDIR /go/src/practice-4
COPY . .

RUN go test ./...
ENV CGO_ENABLED=0
RUN go install ./cmd/...

FROM alpine:latest
WORKDIR /opt/practice-4
COPY --from=build /go/bin/* /opt/practice-4/
COPY entry.sh /opt/practice-4/

# Додамо ці команди:
RUN chmod +x /opt/practice-4/entry.sh && \
    chmod +x /opt/practice-4/server && \
    apk add --no-cache bash

ENTRYPOINT ["/opt/practice-4/entry.sh"]
CMD ["server"]