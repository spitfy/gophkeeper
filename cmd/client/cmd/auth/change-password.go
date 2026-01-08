// cmd/client/cmd/auth/change-password.go
package auth

import (
	"fmt"
	"gophkeeper/internal/app/client"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gophkeeper/internal/domain/user"
)

var changePasswordCmd = &cobra.Command{
	Use:   "change-password",
	Short: "Изменить пароль пользователя",
	Long: `Изменение пароля пользователя на сервере GophKeeper.
	
При смене пароля все данные будут перешифрованы с новым мастер-ключом.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		fmt.Println("=== Смена пароля ===")
		fmt.Println()

		// Запрашиваем текущий пароль
		fmt.Print("Текущий пароль: ")
		currentPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		// Запрашиваем новый пароль
		fmt.Print("Новый пароль: ")
		newPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		fmt.Print("Повторите новый пароль: ")
		newPasswordConfirm, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		if string(newPassword) != string(newPasswordConfirm) {
			return fmt.Errorf("новые пароли не совпадают")
		}

		if len(newPassword) < 8 {
			return fmt.Errorf("новый пароль должен содержать минимум 8 символов")
		}

		// Запрашиваем мастер-пароль для перешифровки
		fmt.Print("Мастер-пароль (для перешифровки данных): ")
		masterPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения мастер-пароля: %w", err)
		}
		fmt.Println()

		// Выполняем смену пароля
		fmt.Println("Смена пароля...")
		err = app.ChangePassword(cmd.Context(), user.ChangePasswordRequest{
			CurrentPassword: string(currentPassword),
			NewPassword:     string(newPassword),
			MasterPassword:  string(masterPassword),
		})
		if err != nil {
			return fmt.Errorf("ошибка смены пароля: %w", err)
		}

		fmt.Println()
		fmt.Println("✅ Пароль успешно изменен!")
		fmt.Println("Все данные перешифрованы с новым мастер-ключом.")

		return nil
	},
}
