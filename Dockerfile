FROM golang:alpine AS builder

# Install system dependencies needed for compiling if any (none needed for standard Go, but git is useful)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application binary for target OS Linux and architecture amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o begmt2 main.go


# --- Final Stage ---
FROM alpine:latest

# Install tzdata for timezone (e.g. Asia/Jakarta) and ca-certificates for outbound HTTPS calls (e.g. Mail/SendGrid/Resend)
RUN apk add --no-cache tzdata ca-certificates

# Set the timezone to Jakarta by default (overrideable via environment variables)
ENV TZ=Asia/Jakarta

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/begmt2 .

# Copy runtime assets needed by the application (PDF generation templates/logos)
COPY --from=builder /app/footer_surat.png .
COPY --from=builder /app/kop_surat.png .

# Create the uploads directory for attachments
RUN mkdir -p uploads

# Expose the default port
EXPOSE 8080

# Command to run the application
CMD ["./begmt2"]
