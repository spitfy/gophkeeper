// cmd/client/cmd/auth/login.go
package auth

import (
	"context"
	"fmt"
	"gophkeeper/internal/app/client"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gophkeeper/internal/domain/user"
)

var (
	rememberMe bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Войти в систему GophKeeper",
	Long: `Аутентификация на сервере GophKeeper.
	
После входа токен сохраняется локально для последующих операций.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		fmt.Println("=== Вход в систему ===")
		fmt.Println()

		// Запрашиваем email
		fmt.Print("Email: ")
		var email string
		fmt.Scanln(&email)

		// Запрашиваем пароль
		fmt.Print("Пароль: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		// Запрашиваем мастер-пароль, если нужно расшифровать локальные данные
		if app.HasLocalData() {
			fmt.Print("Мастер-пароль (для расшифровки локальных данных): ")
			masterPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
			}
			fmt.Println()

			if err := app.UnlockMasterKey(string(masterPassword)); err != nil {
				return fmt.Errorf("неверный мастер-пароль: %w", err)
			}
		}

		// Выполняем вход
		fmt.Println("Аутентификация...")
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		token, err := app.Login(ctx, user.BaseRequest{
			Login:    email,
			Password: string(password),
		})
		if err != nil {
			return fmt.Errorf("ошибка аутентификации: %w", err)
		}

		// Сохраняем токен
		if rememberMe {
			if err := app.SaveToken(token); err != nil {
				return fmt.Errorf("ошибка сохранения токена: %w", err)
			}
		}

		fmt.Println()
		fmt.Println("✅ Вход выполнен успешно!")

		// Синхронизируем данные
		fmt.Println("Синхронизация данных...")
		if err := app.Sync(ctx); err != nil {
			fmt.Printf("⚠️  Предупреждение: ошибка синхронизации: %v\n", err)
			fmt.Println("Вы можете продолжить работу в офлайн-режиме")
		} else {
			fmt.Println("✓ Данные синхронизированы")
		}

		return nil
	},
}

func init() {
	loginCmd.Flags().BoolVarP(&rememberMe, "remember", "r", false, "запомнить меня (сохранить токен)")
}
