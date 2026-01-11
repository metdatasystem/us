FROM golang:1.25.5 AS build

LABEL description="A service supporting the parsing of AWIPS products into the US MDS."

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . ./

# Build with CGO enabled
RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd/parse/awips -o /app/awips

ENTRYPOINT [ "/app/awips", "server" ]

