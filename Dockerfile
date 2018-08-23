FROM golang:alpine as builder

COPY . /go/src/github.com/springernature/halfpipe-deploy-resource
WORKDIR /go/src/github.com/springernature/halfpipe-deploy-resource

ENV CF_TAR_URL "https://packages.cloudfoundry.org/stable?release=linux64-binary&version=6.38.0&source=github-rel"
RUN wget -qO- ${CF_TAR_URL} | tar xvz -C /bin > /dev/null

# This is present as we create a build environment in the Concourse task
# 'Create temp folder with both resource src and plugin from release'
RUN cf install-plugin halfpipe_cf_plugin_linux -f

RUN go build -o /opt/resource/check cmd/check/check.go
RUN go build -o /opt/resource/out cmd/out/out.go
RUN go build -o /opt/resource/in cmd/in/in.go

FROM golang:alpine AS resource
RUN apk add --no-cache bash tzdata ca-certificates jq
COPY --from=builder /opt/resource/* /opt/resource/
COPY --from=builder /bin/cf /bin/cf
COPY --from=builder /root/.cf /root/.cf

RUN chmod +x /opt/resource/*

FROM resource
