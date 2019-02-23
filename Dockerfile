ARG DBTS_PATH=/go/src/github.com/cs3238-tsuzu/dash-button-tradfri-switch

FROM golang:1.11 AS build

ENV GO111MODULE=on
COPY . ${DBTS_PATH}
WORKDIR ${DBTS_PATH}

RUN apt-get update && \
    apt-get install -y libpcap0.8-dev && \
    cd canopus/openssl && \
    ./config && \
    make -j2 && \
    touch ../go.mod

RUN go build -o main --ldflags '-extldflags "-lpthread -static"'

FROM scratch
ARG DBTS_PATH
COPY --from=build ${DBTS_PATH}/main /bin/main

ENTRYPOINT [ "/bin/main" ]
