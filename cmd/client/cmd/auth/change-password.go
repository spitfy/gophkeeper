// cmd/client/cmd/auth/change-password.go
package auth

import (
	"fmt"

	"github.com/spf13/cobra"
)

// TODO: Реализовать на сервере эндпоинт /user/change-password
var ChangePasswordCmd = &cobra.Command{
	Use:   "change-password",
	Short: "Изменить пароль пользователя (не реализовано)",
	Long: `Изменение пароля пользователя на сервере GophKeeper.
	
При смене пароля все данные будут перешифрованы с новым мастер-ключом.

ВНИМАНИЕ: Функция временно недоступна. Требуется реализация на сервере.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("функция смены пароля временно недоступна. Требуется реализация на сервере")
	},
}
