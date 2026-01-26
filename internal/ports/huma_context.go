package ports

import "github.com/danielgtaylor/huma/v2"

type HumaContext interface {
	huma.Context // или явный список методов, которые ты реально зовёшь
}
