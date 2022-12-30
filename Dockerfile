# syntax=docker/dockerfile:1

## Build go application
FROM golang:1.19-buster AS build

WORKDIR /

COPY . ./
RUN go mod download
RUN go build -o /app

## Build frontend
FROM node AS build-frontend

WORKDIR /

COPY . ./
RUN yarn
RUN yarn build

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /app /app
COPY --from=build /config.json /config.json

USER nonroot:nonroot

CMD ["/app"]
