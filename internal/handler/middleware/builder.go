package middleware

import (
	"github.com/danielgtaylor/huma/v2"
)

// Container - обертка над Middlewares с дополнительной функциональностью
type Container struct {
	huma.Middlewares
}

// NewContainer создает новый контейнер для мидлварей
func NewContainer() *Container {
	return &Container{
		Middlewares: make(huma.Middlewares, 0),
	}
}

// Add добавляет одну мидлварь в контейнер
func (mc *Container) Add(middleware func(ctx huma.Context, next func(huma.Context))) {
	mc.Middlewares = append(mc.Middlewares, middleware)
}

// GetAllAndClear возвращает все мидлвари и очищает внутренний список
func (mc *Container) GetAllAndClear() huma.Middlewares {
	result := mc.Middlewares
	mc.Middlewares = nil
	return result
}
