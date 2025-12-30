### Откат миграции на 1 шаг из директории `cmd/server`
`migrate -path ../../migrations -database postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable down 1`
### Старт миграции с 3й позиции 
`migrate -path ../../migrations -database "postgres://postgres:postgres@localhost:5432/gophkeeper?sslmode=disable" force 3`