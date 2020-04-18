FROM golang:1.14-alpine3.11

WORKDIR /go/src/github.com/aoepeople/gitlab-crucible-bridge/

COPY . .

RUN apk --no-cache add curl git
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gitlab-crucible-bridge .

FROM alpine:3.11
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=0 /go/src/github.com/aoepeople/gitlab-crucible-bridge/gitlab-crucible-bridge .

CMD ["./gitlab-crucible-bridge"]
