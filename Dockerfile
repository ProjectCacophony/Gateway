# golang image to build the service
FROM golang:1.11-stretch as builder

# install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
  && rm -rf /var/lib/apt/lists/*

# workdir /src because we have to be out of go path to use go modules
WORKDIR /src

# copy source code
COPY ./ ./

# change GOPATH to ./go as we copy the cache into there on CI
# update PATH to include ./go/bin
ENV GOPATH /src/go
ENV PATH $PATH:/src/go/bin

# ability to inject goproxy
ARG GOPROXY

# download deps
RUN go mod download

# build binary
RUN make all

# ubuntu image to run service
FROM ubuntu:bionic

# install dependencies, and clean up
RUN apt-get update && TERM=linux DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
  && rm -rf /var/lib/apt/lists/*

# copy binary from builder step
COPY --from=builder /src/bin/linux.amd64 /

# copy assets into image
COPY ./assets ./assets

# expose http server port
#EXPOSE 8000

# switch to user without permissions
USER nobody

# run the binary
ENTRYPOINT [ "/gateway" ]
