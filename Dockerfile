FROM golang:1.26 AS build
WORKDIR /src

ARG GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${GOOS} go build -trimpath -ldflags="-s -w" -o /out/ddns ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/ddns /ddns
USER nonroot:nonroot
ENTRYPOINT ["/ddns"]
