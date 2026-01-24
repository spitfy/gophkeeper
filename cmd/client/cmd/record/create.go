// cmd/client/cmd/record/create.go
package record

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"gophkeeper/cmd/client/cmd/types"
	"gophkeeper/internal/app/client"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
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

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Создать новую запись",
	Long: `Создание новой защищенной записи.
	
Поддерживаемые типы записей:
- password - логин и пароль
- note     - текстовая заметка
- card     - данные банковской карты
- file     - бинарный файл`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value(types.ClientAppKey).(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		if !app.IsMasterKeyUnlocked() {
			fmt.Println("❌ Мастер-ключ заблокирован")
			fmt.Println()
			fmt.Println("Для создания записей необходимо разблокировать мастер-ключ.")
			fmt.Println("Выполните команду: gophkeeper unlock")
			return fmt.Errorf("мастер-ключ заблокирован")
		}

		if recordType == "" {
			fmt.Println("Выберите тип записи:")
			fmt.Println("1. Пароль (логин/пароль)")
			fmt.Println("2. Текстовая заметка")
			fmt.Println("3. Банковская карта")
			fmt.Println("4. Файл")
			fmt.Print("Ваш выбор [1-4]: ")

			var choice string
			_, _ = fmt.Scanln(&choice)

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

		// Создаем запись в зависимости от типа
		var recordID int
		var err error

		switch strings.ToLower(recordType) {
		case "password":
			recordID, err = createPasswordRecord(cmd, app)
		case "note":
			recordID, err = createNoteRecord(cmd, app)
		case "card":
			recordID, err = createCardRecord(cmd, app)
		case "file":
			recordID, err = createFileRecord(cmd, app)
		default:
			return fmt.Errorf("неподдерживаемый тип записи: %s", recordType)
		}

		if err != nil {
			return fmt.Errorf("ошибка создания записи: %w", err)
		}

		fmt.Println()
		fmt.Printf("✅ Запись '%s' успешно создана! (ID: %d)\n", recordName, recordID)

		return nil
	},
}

func createPasswordRecord(cmd *cobra.Command, app *client.App) (int, error) {
	if username == "" {
		fmt.Print("Логин/Email: ")
		_, err := fmt.Scanln(&username)
		if err != nil {
			return 0, err
		}
	}

	if password == "" {
		fmt.Print("Пароль (оставьте пустым для генерации): ")
		var pass string
		_, err := fmt.Scanln(&pass)
		if err != nil {
			return 0, err
		}

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
		_, err := fmt.Scanln(&url)
		if err != nil {
			return 0, err
		}
	}

	req := client.CreateLoginRequest{
		Username: username,
		Password: password,
		Title:    recordName,
		Resource: url,
		Notes:    description,
	}

	fmt.Println("Создание записи...")
	return app.CreateLoginRecord(cmd.Context(), req)
}

func createNoteRecord(cmd *cobra.Command, app *client.App) (int, error) {
	if noteContent == "" {
		fmt.Println("Введите текст заметки (Ctrl+D для завершения):")
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		noteContent = strings.Join(lines, "\n")
	}

	req := client.CreateTextRequest{
		Content: noteContent,
		Title:   recordName,
	}

	fmt.Println("Создание записи...")
	return app.CreateTextRecord(cmd.Context(), req)
}

func createCardRecord(cmd *cobra.Command, app *client.App) (int, error) {
	if cardNumber == "" {
		fmt.Print("Номер карты: ")
		_, err := fmt.Scanln(&cardNumber)
		if err != nil {
			return 0, err
		}
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
		_, _ = fmt.Scanln(&expiryDate)
	}

	if cvv == "" {
		fmt.Print("CVV: ")
		_, _ = fmt.Scanln(&cvv)
	}

	// Разбираем срок действия
	parts := strings.Split(expiryDate, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("неверный формат срока действия. Используйте MM/YY")
	}

	req := client.CreateCardRequest{
		CardNumber:  cardNumber,
		CardHolder:  cardHolder,
		ExpiryMonth: parts[0],
		ExpiryYear:  "20" + parts[1], // Преобразуем YY в 20YY
		CVV:         cvv,
		Title:       recordName,
		Notes:       description,
	}

	fmt.Println("Создание записи...")
	return app.CreateCardRecord(cmd.Context(), req)
}

func createFileRecord(cmd *cobra.Command, app *client.App) (int, error) {
	if filePath == "" {
		fmt.Print("Путь к файлу: ")
		_, _ = fmt.Scanln(&filePath)
	}

	// Читаем файл
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("ошибка чтения файла: %w", err)
	}

	// Кодируем в base64
	encodedData := make([]byte, len(data)*2)
	n := 0
	for _, b := range data {
		encodedData[n] = b
		n++
	}

	req := client.CreateBinaryRequest{
		Data:        string(data),
		Filename:    filepath.Base(filePath),
		Title:       recordName,
		Description: description,
	}

	fmt.Println("Создание записи...")
	return app.CreateBinaryRecord(cmd.Context(), req)
}

func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

func init() {
	CreateCmd.Flags().StringVarP(&recordType, "type", "t", "", "тип записи (password, note, card, file)")
	CreateCmd.Flags().StringVarP(&recordName, "name", "n", "", "название записи")
	CreateCmd.Flags().StringVar(&description, "desc", "", "описание записи")

	// Флаги для паролей
	CreateCmd.Flags().StringVar(&username, "username", "", "логин/email")
	CreateCmd.Flags().StringVar(&password, "password", "", "пароль")
	CreateCmd.Flags().StringVar(&url, "url", "", "URL сайта")

	// Флаги для заметок
	CreateCmd.Flags().StringVar(&noteContent, "content", "", "содержимое заметки")

	// Флаги для карт
	CreateCmd.Flags().StringVar(&cardNumber, "card-number", "", "номер карты")
	CreateCmd.Flags().StringVar(&cardHolder, "card-holder", "", "держатель карты")
	CreateCmd.Flags().StringVar(&expiryDate, "expiry", "", "срок действия (MM/YY)")
	CreateCmd.Flags().StringVar(&cvv, "cvv", "", "CVV код")

	// Флаги для файлов
	CreateCmd.Flags().StringVar(&filePath, "file", "", "путь к файлу")
}
