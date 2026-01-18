// cmd/client/cmd/init.go
package cmd

import (
	"fmt"
	"os"

	"gophkeeper/cmd/client/cmd/auth"
	"gophkeeper/cmd/client/cmd/record"
	"gophkeeper/cmd/client/cmd/sync"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Инициализировать клиент GophKeeper",
	Long: `Команда init выполняет первоначальную настройку клиента:
	1. Создает мастер-ключ для шифрования данных
	2. Настраивает директорию для хранения данных
	3. Проверяет соединение с сервером
	
Мастер-ключ защищает все ваши данные. Убедитесь, что выбрали надежный пароль
и сохранили его в безопасном месте. Без мастер-ключа восстановить данные невозможно.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Проверяем, не инициализирован ли уже клиент
		if app.IsInitialized() {
			fmt.Println("Клиент уже инициализирован.")
			return nil
		}

		fmt.Println("=== Инициализация GophKeeper ===")
		fmt.Println()

		// Запрашиваем мастер-пароль
		fmt.Print("Введите мастер-пароль: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("ошибка чтения пароля: %w", err)
		}
		fmt.Println()

		fmt.Print("Повторите мастер-пароль: ")
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

		// Инициализируем мастер-ключ
		fmt.Println("Создание мастер-ключа...")
		if err := app.InitMasterKey(string(password)); err != nil {
			return fmt.Errorf("ошибка создания мастер-ключа: %w", err)
		}

		// Проверяем соединение с сервером
		fmt.Println("Проверка соединения с сервером...")
		if err := app.CheckConnection(); err != nil {
			fmt.Printf("⚠️  Предупреждение: не удалось подключиться к серверу: %v\n", err)
			fmt.Println("Вы можете работать в офлайн-режиме, но синхронизация будет недоступна.")
		} else {
			fmt.Println("✓ Соединение с сервером установлено")
		}

		// Создаем базовую структуру
		fmt.Println("Создание структуры данных...")
		if err := app.InitStorage(); err != nil {
			return fmt.Errorf("ошибка инициализации хранилища: %w", err)
		}

		fmt.Println()
		fmt.Println("✅ Инициализация успешно завершена!")
		fmt.Println()
		fmt.Println("Что дальше:")
		fmt.Println("1. Зарегистрируйтесь на сервере: gophkeeper auth register")
		fmt.Println("2. Войдите в систему: gophkeeper auth login")
		fmt.Println("3. Создайте первую запись: gophkeeper record create")

		return nil
	},
}

func init() {
	// Добавляем команды аутентификации
	rootCmd.AddCommand(auth.AuthCmd)
	auth.AuthCmd.AddCommand(auth.RegisterCmd)
	auth.AuthCmd.AddCommand(auth.LoginCmd)
	auth.AuthCmd.AddCommand(auth.ChangePasswordCmd)

	// Добавляем команды работы с записями
	rootCmd.AddCommand(record.RecordCmd)
	record.RecordCmd.AddCommand(record.CreateCmd)
	record.RecordCmd.AddCommand(record.GetCmd)
	record.RecordCmd.AddCommand(record.ListCmd)

	rootCmd.AddCommand(sync.SyncCmd)
}
