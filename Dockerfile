FROM golang:1.23-bookworm AS builder

COPY . /build
WORKDIR /build

ENV CF_TAR_URL_V6="https://packages.cloudfoundry.org/stable?release=linux64-binary&version=6.53.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL_V6} | tar xvz -C /bin > /dev/null
RUN mv /bin/cf /bin/cf6

ENV CF_TAR_URL_V7="https://packages.cloudfoundry.org/stable?release=linux64-binary&version=7.8.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL_V7} | tar xvz -C /bin > /dev/null

ENV CF_TAR_URL_V8="https://packages.cloudfoundry.org/stable?release=linux64-binary&version=8.11.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL_V8} | tar xvz -C /bin > /dev/null

RUN go test ./...
RUN CGO_ENABLED=0 go build -o /opt/resource/check cmd/check/check.go
RUN CGO_ENABLED=0 go build -o /opt/resource/out cmd/out/out.go
RUN CGO_ENABLED=0 go build -o /opt/resource/in cmd/in/in.go
RUN chmod +x /opt/resource/*

ADD .git/ref /opt/resource/builtWithRef

FROM gcr.io/distroless/static-debian12
COPY --from=builder /opt/resource/* /opt/resource/
COPY --from=builder /bin/cf6 /bin/cf6
COPY --from=builder /bin/cf7 /bin/cf7
COPY --from=builder /bin/cf8 /bin/cf8
ENV TERM=xterm-256color

ENTRYPOINT ["/opt/resource/out"]
