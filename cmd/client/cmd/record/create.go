// cmd/client/cmd/record/create.go
package record

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gophkeeper/internal/app/client"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gophkeeper/internal/domain/record"
)

var (
	recordType  string
	recordName  string
	description string
	username    string
	password    string
	url         string
	noteContent string
	cardNumber  string
	cardHolder  string
	expiryDate  string
	cvv         string
	filePath    string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Создать новую запись",
	Long: `Создание новой защищенной записи.
	
Поддерживаемые типы записей:
- password - логин и пароль
- note     - текстовая заметка
- card     - данные банковской карты
- file     - бинарный файл`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		// Если тип записи не указан, спрашиваем
		if recordType == "" {
			fmt.Println("Выберите тип записи:")
			fmt.Println("1. Пароль (логин/пароль)")
			fmt.Println("2. Текстовая заметка")
			fmt.Println("3. Банковская карта")
			fmt.Println("4. Файл")
			fmt.Print("Ваш выбор [1-4]: ")

			var choice string
			fmt.Scanln(&choice)

			switch choice {
			case "1":
				recordType = "password"
			case "2":
				recordType = "note"
			case "3":
				recordType = "card"
			case "4":
				recordType = "file"
			default:
				return fmt.Errorf("неверный выбор")
			}
		}

		// Определяем тип записи
		var recType record.RecType
		switch strings.ToLower(recordType) {
		case "password":
			recType = record.RecTypeLogin
		case "note":
			recType = record.RecTypeText
		case "card":
			recType = record.RecTypeText
		case "file":
			recType = record.RecTypeBinary
		default:
			return fmt.Errorf("неподдерживаемый тип записи: %s", recordType)
		}

		// Запрашиваем имя записи
		if recordName == "" {
			fmt.Print("Название записи: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				recordName = scanner.Text()
			}
			if recordName == "" {
				return fmt.Errorf("название записи обязательно")
			}
		}

		// Запрашиваем описание
		if description == "" && recType != record.RecTypeBinary {
			fmt.Print("Описание (необязательно, Enter чтобы пропустить): ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				description = scanner.Text()
			}
		}

		// Собираем данные в зависимости от типа
		var data []byte
		var err error

		switch recType {
		case record.RecTypeLogin:
			data, err = createPasswordData()
		case record.RecTypeText:
			data, err = createNoteData()
		case record.RecTypeCard:
			data, err = createCardData()
		case record.RecTypeBinary:
			data, err = createFileData()
		}
		if err != nil {
			return err
		}

		// Создаем запись
		rec := &record.Record{
			Type: recType,
			Metadata: record.Metadata{
				Name:        recordName,
				Description: description,
			},
			EncryptedData: data,
		}

		// Сохраняем запись
		fmt.Println("Создание записи...")
		err = app.CreateRecord(cmd.Context(), rec)
		if err != nil {
			return fmt.Errorf("ошибка создания записи: %w", err)
		}

		fmt.Println()
		fmt.Printf("✅ Запись '%s' успешно создана!\n", recordName)

		return nil
	},
}

func createPasswordData() ([]byte, error) {
	if username == "" {
		fmt.Print("Логин/Email: ")
		fmt.Scanln(&username)
	}

	if password == "" {
		fmt.Print("Пароль (оставьте пустым для генерации): ")
		var pass string
		fmt.Scanln(&pass)

		if pass == "" {
			// Генерируем пароль
			password = generatePassword(12)
			fmt.Printf("Сгенерирован пароль: %s\n", password)
		} else {
			password = pass
		}
	}

	if url == "" {
		fmt.Print("URL (необязательно): ")
		fmt.Scanln(&url)
	}

	passwordData := map[string]string{
		"username": username,
		"password": password,
		"url":      url,
	}

	return json.Marshal(passwordData)
}

func createNoteData() ([]byte, error) {
	if noteContent == "" {
		fmt.Println("Введите текст заметки (Ctrl+D для завершения):")
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		noteContent = strings.Join(lines, "\n")
	}

	return []byte(noteContent), nil
}

func createCardData() ([]byte, error) {
	if cardNumber == "" {
		fmt.Print("Номер карты: ")
		fmt.Scanln(&cardNumber)
	}

	if cardHolder == "" {
		fmt.Print("Держатель карты: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			cardHolder = scanner.Text()
		}
	}

	if expiryDate == "" {
		fmt.Print("Срок действия (MM/YY): ")
		fmt.Scanln(&expiryDate)
	}

	if cvv == "" {
		fmt.Print("CVV: ")
		fmt.Scanln(&cvv)
	}

	cardData := map[string]string{
		"number":     cardNumber,
		"holder":     cardHolder,
		"expiryDate": expiryDate,
		"cvv":        cvv,
	}

	return json.Marshal(cardData)
}

func createFileData() ([]byte, error) {
	if filePath == "" {
		fmt.Print("Путь к файлу: ")
		fmt.Scanln(&filePath)
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Сохраняем оригинальное имя файла в метаданных
	if description == "" {
		description = filepath.Base(filePath)
	}

	return data, nil
}

func generatePassword(length int) string {
	// Простой генератор паролей
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func init() {
	createCmd.Flags().StringVarP(&recordType, "type", "t", "", "тип записи (password, note, card, file)")
	createCmd.Flags().StringVarP(&recordName, "name", "n", "", "название записи")
	createCmd.Flags().StringVar(&description, "desc", "", "описание записи")

	// Флаги для паролей
	createCmd.Flags().StringVar(&username, "username", "", "логин/email")
	createCmd.Flags().StringVar(&password, "password", "", "пароль")
	createCmd.Flags().StringVar(&url, "url", "", "URL сайта")

	// Флаги для заметок
	createCmd.Flags().StringVar(&noteContent, "content", "", "содержимое заметки")

	// Флаги для карт
	createCmd.Flags().StringVar(&cardNumber, "card-number", "", "номер карты")
	createCmd.Flags().StringVar(&cardHolder, "card-holder", "", "держатель карты")
	createCmd.Flags().StringVar(&expiryDate, "expiry", "", "срок действия (MM/YY)")
	createCmd.Flags().StringVar(&cvv, "cvv", "", "CVV код")

	// Флаги для файлов
	createCmd.Flags().StringVar(&filePath, "file", "", "путь к файлу")
}
