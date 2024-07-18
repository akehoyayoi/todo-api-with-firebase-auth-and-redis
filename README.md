# Prepare

1. Firebaseコンソールで新しいプロジェクトを作成します。
1. プロジェクト設定から、「サービスアカウント」タブに移動し、新しいサービスアカウントキーを生成してダウンロードします。
1. Firebase Authenticationを有効にし、必要なサインインメソッド（例えば、メール/パスワード、Google、Facebookなど）を設定します。

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
ID_TOKEN=$(curl 'https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key="<YOUR_KEY>"' \
-H 'Content-Type: application/json' \
--data-binary '{"email":"<YOUR_ADDRESS>"","password":"<YOUR_PASSWORD","returnSecureToken":true}' | jq -r .idToken)
```

Refs: https://qiita.com/kazakago/items/892a8c5df76a912f1d82




## POST /api/todos

```
curl -X POST -H "Authorization: Bearer ${ID_TOKEN}" -H "Content-Type: application/json" -d '{"text": "New Task", "lat": 35.6895, "lng": 139.6917}' http://localhost:8079/api/todos
```

## PUT /api/todos/:id

```
curl -X PUT -H "Authorization: Bearer ${ID_TOKEN}" -H "Content-Type: application/json" -d '{"text": "Updated Task", "done": true, "lat": 35.6895, "lng": 139.6918}' http://localhost:8079/api/todos/<RETURNED_ID>
```

## DELETE /api/todos/:id
```
curl -X DELETE -H "Authorization: Bearer ${ID_TOKEN}" http://localhost:8079/api/todos/<RETURNED_ID>
```

## GET /api/todos/search
```
curl -H "Authorization: Bearer ${ID_TOKEN}" "http://localhost:8079/api/todos/search?lat=35.6895&lng=139.6917&radius=10"
```
