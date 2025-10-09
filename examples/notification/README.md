# Notification Example

## How to Use

### build notification service

```bash
$ tbls doc
$ go get
$ go build ./cmd/notification
```

### launch DB

```bash
$ docker compose up
```

### launch notification service

```bash
# init database (exec once)
$ ./notification -init -dsn postgres://notification:notification@localhost:5432/notification?sslmode=disable
$ ./notification -dsn postgres://notification:notification@localhost:5432/notification?sslmode=disable
```

### launch frontend

```bash
$ cd frontend
$ npm install
$ npm run dev
```
