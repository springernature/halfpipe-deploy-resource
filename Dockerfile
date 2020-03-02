FROM golang:1.13-buster as builder

COPY . /build
WORKDIR /build

ENV CF_TAR_URL "https://packages.cloudfoundry.org/stable?release=linux64-binary&version=7.0.0-beta.30&source=github-rel"
RUN wget -qO- ${CF_TAR_URL} | tar xvz -C /bin > /dev/null
RUN mv /bin/cf7 /bin/cf

# This is present as we create a build environment in the Concourse task
# 'Create temp folder with both resource src and plugin from release'
RUN cf install-plugin halfpipe_cf_plugin_linux -f

RUN go test ./...
RUN go build -o /opt/resource/check cmd/check/check.go
RUN go build -o /opt/resource/out cmd/out/out.go
RUN go build -o /opt/resource/in cmd/in/in.go

RUN chmod +x /opt/resource/*

FROM golang:alpine AS resource
RUN apk add --no-cache bash tzdata ca-certificates jq libc6-compat
COPY --from=builder /opt/resource/* /opt/resource/
COPY --from=builder /bin/cf /bin/cf
COPY --from=builder /root/.cf /root/.cf

FROM resource
