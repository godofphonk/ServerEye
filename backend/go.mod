module github.com/servereye/servereye/backend

go 1.24.0

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.0
	github.com/lib/pq v1.10.9
	github.com/servereye/servereye v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/net v0.48.0
)

replace github.com/servereye/servereye => ../

require (
	github.com/google/uuid v1.4.0 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
