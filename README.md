# csv2sql


csvは
`csv2sql/csv/kdb.csv`

### 環境変数を設定
```
export SYLMS_POSTGRES_DB=sylms
export SYLMS_POSTGRES_USER=sylms
export SYLMS_POSTGRES_PASSWORD=sylms
export SYLMS_POSTGRES_HOST=127.0.0.1
export SYLMS_POSTGRES_PORT=5432
export SYLMS_CSV_YEAR=2021
```

### データベースを起動
```
docker-compose -f docker-compose.db.yml up -d
```

### ビルド
```
go build -o ./build && ./build
```


