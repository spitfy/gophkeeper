// cmd/client/cmd/auth/register.go
package auth

import (
	"fmt"
	"gophkeeper/cmd/client/cmd/types"
	"gophkeeper/internal/app/client"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gophkeeper/internal/domain/user"
)

var RegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Зарегистрировать нового пользователя",
	Long: `Регистрация нового пользователя на сервере GophKeeper.
	
После регистрации вы сможете синхронизировать данные между устройствами.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Получаем приложение из контекста
		app := cmd.Context().Value(types.ClientAppKey).(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		fmt.Println("=== Регистрация нового пользователя ===")
		fmt.Println()

		// Запрашиваем email
		fmt.Print("Login: ")
		var login string
		_, _ = fmt.Scanln(&login)

		// Запрашиваем пароль
		fmt.Print("Пароль: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		fmt.Print("Повторите пароль: ")
		passwordConfirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		if string(password) != string(passwordConfirm) {
			return fmt.Errorf("пароли не совпадают")
		}

		if len(password) < 8 {
			return fmt.Errorf("пароль должен содержать минимум 8 символов")
		}

		// Регистрируем пользователя
		fmt.Println("Регистрация...")
		err = app.Register(cmd.Context(), user.BaseRequest{
			Login:    login,
			Password: string(password),
		})
		if err != nil {
			return fmt.Errorf("ошибка регистрации: %w", err)
		}

		fmt.Println()
		fmt.Println("✅ Регистрация успешно завершена!")
		fmt.Println("Теперь вы можете войти в систему: gophkeeper auth login")

		return nil
	},
}
