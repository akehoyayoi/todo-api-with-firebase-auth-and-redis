# Prepare

Firebaseコンソールで新しいプロジェクトを作成します。
プロジェクト設定から、「サービスアカウント」タブに移動し、新しいサービスアカウントキーを生成してダウンロードします。
Firebase Authenticationを有効にし、必要なサインインメソッド（例えば、メール/パスワード、Google、Facebookなど）を設定します。

# Build&Execute

```
# Dockerイメージのビルド
docker-compose build

# Dockerコンテナの実行
docker-compose up
```

# Test

## For getting YOUR_ID_TOKEN

```
curl 'https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=<YOUR_KEY>' \
-H 'Content-Type: application/json' \
--data-binary '{"email":"<YOUR_ADDRESS>","password":"<YOUR_PASSWORD>","returnSecureToken":true}'
```

Refs: https://qiita.com/kazakago/items/892a8c5df76a912f1d82



## GET /api/todos

```
curl -H "Authorization: Bearer <YOUR_ID_TOKEN>" http://localhost:8079/api/todos
```

## POST /api/todos

```
curl -X POST -H "Authorization: Bearer <YOUR_ID_TOKEN>" -H "Content-Type: application/json" -d '{"text": "New Task"}' http://localhost:8079/api/todos
```

## PUT /api/todos/:id

```
curl -X PUT -H "Authorization: Bearer <YOUR_ID_TOKEN>" -H "Content-Type: application/json" -d '{"text": "Updated Task", "done": true}' http://localhost:8079/api/todos/1
```

## DELETE /api/todos/:id
```
curl -X DELETE -H "Authorization: Bearer <YOUR_ID_TOKEN>" http://localhost:8079/api/todos/1
```
