# ベースイメージとしてGoの公式イメージを使用
FROM golang:1.20-alpine

# ビルドに必要なツールをインストール
RUN apk add --no-cache git

# 作業ディレクトリを設定
WORKDIR /app

# Go Modulesのキャッシュを有効にするためにgo.modとgo.sumをコピー
COPY go.mod go.sum ./
RUN go mod download

# アプリケーションのソースコードをコピー
COPY . .

# アプリケーションをビルド
RUN go build -o main .

# サービスアカウントキーをコピー
COPY serviceAccountKey.json /app/serviceAccountKey.json

# アプリケーションを起動
CMD ["./main"]

# ポート8080を公開
EXPOSE 8080

