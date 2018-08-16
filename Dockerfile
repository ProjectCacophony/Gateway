ARG UPSTREAM_IMAGE="golang:1.10"
FROM "${UPSTREAM_IMAGE}"

ARG SRC_DIR="gitlab.com/Cacophony/Gateway"
WORKDIR "/go/src/${SRC_DIR}"

COPY . .

# Install discordgo, and checkout the 'develop' branch
RUN go get -v github.com/bwmarrin/discordgo \
  && cd "${GOPATH}/src/github.com/bwmarrin/discordgo" \
  && git checkout -f develop

RUN go get -v ./...