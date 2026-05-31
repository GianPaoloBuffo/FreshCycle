FROM golang:1.26-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /src/bin/freshcycle-api ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
		tesseract-ocr \
		tesseract-ocr-eng \
		tesseract-ocr-spa \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /src/bin/freshcycle-api ./freshcycle-api

ENV OCR_TESSERACT_PATH=/usr/bin/tesseract
ENV OCR_LANGUAGES=eng+spa

CMD ["./freshcycle-api"]
