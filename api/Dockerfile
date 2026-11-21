# golang:1.27.4-bookworm
FROM golang@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS build
ENV GOTOOLCHAIN=local CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/happybabytracker-api ./cmd/happybabytracker-api \
    && mkdir /out/data

# gcr.io/distroless/static-debian12:nonroot
FROM gcr.io/distroless/static-debian12@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
LABEL org.opencontainers.image.source="https://github.com/davcs86/happybabytracker" \
      org.opencontainers.image.licenses="MIT"
COPY --from=build /out/happybabytracker-api /happybabytracker-api
COPY --from=build --chown=10001:10001 /out/data /data
USER 10001:10001
ENV DATA_DIR=/data
# Two concurrent argon2 hashes (~38 MiB live) must stay under the 60 MiB ceiling in a 96m container.
ENV GOMEMLIMIT=56MiB
VOLUME /data
EXPOSE 9210
ENTRYPOINT ["/happybabytracker-api"]
