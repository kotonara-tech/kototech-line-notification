# syntax=docker/dockerfile:1

# ビルド用。go.mod / go.sum を先にコピーして、依存のダウンロードをキャッシュさせる
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/notifier ./cmd/notifier

# 実行用。静的ビルドしたバイナリ 1 つだけを、非 root で動かす
# ROLE などの環境変数は、イメージには焼き込まず、実行時(Terraform)に与える
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/notifier /notifier
ENTRYPOINT ["/notifier"]
