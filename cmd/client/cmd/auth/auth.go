package auth

import (
	"github.com/spf13/cobra"
)

// AuthCmd - родительская команда для всех операций с авторизацей пользователя
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Управление пользователем",
	Long:  `Авторизация, регистрация, изменение пароля.`,
}
