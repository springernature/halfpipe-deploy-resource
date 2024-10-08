FROM golang:1.23-alpine as builder

COPY . /build
WORKDIR /build

ENV CF_TAR_URL_V6 "https://packages.cloudfoundry.org/stable?release=linux64-binary&version=6.53.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL_V6} | tar xvz -C /bin > /dev/null
RUN mv /bin/cf /bin/cf6

ENV CF_TAR_URL_V7 "https://packages.cloudfoundry.org/stable?release=linux64-binary&version=7.5.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL_V7} | tar xvz -C /bin > /dev/null

ENV CF_TAR_URL_V8 "https://packages.cloudfoundry.org/stable?release=linux64-binary&version=8.5.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL_V8} | tar xvz -C /bin > /dev/null

RUN go test ./...
RUN go build -o /opt/resource/check cmd/check/check.go
RUN go build -o /opt/resource/out cmd/out/out.go
RUN go build -o /opt/resource/in cmd/in/in.go
RUN chmod +x /opt/resource/*

ADD .git/ref /opt/resource/builtWithRef

FROM golang:1.23-alpine AS resource
RUN apk add --no-cache bash tzdata ca-certificates jq libc6-compat
ENV TERM xterm-256color
COPY --from=builder /opt/resource/* /opt/resource/
COPY --from=builder /bin/cf6 /bin/cf6
COPY --from=builder /bin/cf7 /bin/cf7
COPY --from=builder /bin/cf8 /bin/cf8

FROM resource
ENTRYPOINT ["/opt/resource/out"]
