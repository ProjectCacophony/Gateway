# ubuntu image to run service
FROM ubuntu:bionic

# install dependencies, and clean up
RUN apt-get update && TERM=linux DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
  && rm -rf /var/lib/apt/lists/*

# copy binary into image
COPY ./bin/linux.amd64/gateway /gateway

# copy assets into image
COPY ./assets ./assets

# expose http server port
EXPOSE 8000

# switch to user without permissions
USER nobody

# run the binary
ENTRYPOINT [ "/gateway" ]
