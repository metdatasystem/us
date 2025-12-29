FROM golang:1.25.4 AS build

LABEL description="A service providing real-time information of various datasets in the US MDS."

# Install GEOS
RUN apt-get update && apt-get install -y libgeos-dev

# Set destination for COPY
WORKDIR /

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . ./

# Build with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o /live

# Use glibc-based image in the final stage
FROM debian:bookworm-slim AS release

WORKDIR /

# Install runtime dependencies
RUN apt-get update && apt-get install -y libgeos-c1v5

COPY --from=build /live /live

ENTRYPOINT [ "/live" ]