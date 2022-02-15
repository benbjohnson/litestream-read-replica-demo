FROM litestream/litestream:pr-321 AS litestream


FROM golang:1.17 as builder

COPY . /src/litestream-read-replica-demo
WORKDIR /src/litestream-read-replica-demo

RUN --mount=type=cache,target=/root/.cache/go-build \
	--mount=type=cache,target=/go/pkg \
	go build -ldflags '-s -w -extldflags "-static"' -tags osusergo,netgo,sqlite_omit_load_extension -o /usr/local/bin/litestream-read-replica-demo .



FROM alpine

COPY --from=builder /usr/local/bin/litestream-read-replica-demo /usr/local/bin/litestream-read-replica-demo
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream

ADD run.sh /run.sh
ADD etc/litestream.primary.yml /etc/litestream.primary.yml
ADD etc/litestream.replica.yml /etc/litestream.replica.yml

RUN apk add bash ca-certificates curl

RUN mkdir -p /data

CMD /run.sh
