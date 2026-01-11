FROM golang:1.25.5 AS build

LABEL description="A service supporting the ingest of AWIPS products into the US MDS."

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd/ingest/awips -o /app/awips

ENTRYPOINT [ "/app/awips", "nwws" ]
