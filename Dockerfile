# golang image to build the service
FROM golang:1.11-stretch as builder

# install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
  && rm -rf /var/lib/apt/lists/*

# workdir /src because we have to be out of go path to use go modules
WORKDIR /src

# fetch dependencies in own step to support caching it
COPY ./go.mod ./go.sum ./
RUN go mod download

# copy source code
COPY ./ ./

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

# expose http server port
#EXPOSE 8000

# run the binary
ENTRYPOINT [ "/gateway" ]