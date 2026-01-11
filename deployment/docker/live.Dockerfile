FROM golang:1.25.5 AS build

LABEL description="A service providing real-time information of various datasets in the US MDS."

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . ./

# Build with CGO enabled
RUN CGO_ENABLED=0 GOOS=linux go build -C ./internal/live -o /app/live

ENTRYPOINT [ "/app/live" ]