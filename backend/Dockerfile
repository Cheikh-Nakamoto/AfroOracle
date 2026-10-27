# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /afrioracle .

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /afrioracle /usr/local/bin/afrioracle
EXPOSE 8000
ENV PORT=8000
ENTRYPOINT ["afrioracle"]
