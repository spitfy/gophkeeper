// cmd/client/cmd/record/get.go
package record

import (
	"encoding/json"
	"fmt"
	"gophkeeper/cmd/client/cmd/clientctx"
	"gophkeeper/internal/app/client"
	"gophkeeper/internal/domain/record"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	showPassword bool
	decrypt      bool
)

var GetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Просмотреть запись",
	Long: `Просмотр содержимого записи по ID.
	
Вы можете указать формат вывода и решить, показывать ли чувствительные данные.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value(clientctx.ClientAppKey).(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		recordID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("неверный ID записи: %w", err)
		}

		// Получаем запись
		rec, err := app.GetRecord(cmd.Context(), recordID)
		if err != nil {
			return fmt.Errorf("ошибка получения записи: %w", err)
		}

		// Если нужна расшифровка, получаем расшифрованные данные
		var decryptedData interface{}
		if decrypt {
			if !app.IsMasterKeyUnlocked() {
				return fmt.Errorf("мастер-ключ заблокирован. Выполните: gophkeeper unlock")
			}
			decryptedData, err = app.GetDecryptedRecord(cmd.Context(), recordID)
			if err != nil {
				return fmt.Errorf("ошибка расшифровки записи: %w", err)
			}
		}

		switch outputFormat {
		case "json":
			return printRecordJSON(rec, decryptedData, showPassword)
		case "yaml":
			return printRecordYAML(rec, decryptedData, showPassword)
		default:
			return printRecordHuman(rec, decryptedData, showPassword)
		}
	},
}

func printRecordHuman(rec *client.LocalRecord, decryptedData interface{}, showPassword bool) error {
	fmt.Printf("ID:          %d\n", rec.ID)
	fmt.Printf("Server ID:   %d\n", rec.ServerID)
	fmt.Printf("Тип:         %s\n", rec.Type)

	var meta map[string]interface{}
	if len(rec.Meta) > 0 {
		if err := json.Unmarshal(rec.Meta, &meta); err == nil {
			if title, ok := meta["title"].(string); ok {
				fmt.Printf("Название:    %s\n", title)
			}
		}
	}

	fmt.Printf("Обновлено:   %s\n", rec.LastModified.Format("2006-01-02 15:04:05"))
	fmt.Printf("Версия:      %d\n", rec.Version)
	fmt.Printf("Синхронизирована: %v\n", rec.Synced)
	fmt.Println()

	// Если данные расшифрованы, показываем их
	if decryptedData != nil {
		switch rec.Type {
		case record.RecTypeLogin:
			fmt.Println("=== Данные авторизации ===")
			if dataMap, ok := decryptedData.(map[string]interface{}); ok {
				if username, ok := dataMap["username"].(string); ok {
					fmt.Printf("Логин:       %s\n", username)
				}
				if showPassword {
					if password, ok := dataMap["password"].(string); ok {
						fmt.Printf("Пароль:      %s\n", password)
					}
				} else {
					fmt.Println("Пароль:      ******** (используйте --show-password)")
				}
				if notes, ok := dataMap["notes"].(string); ok && notes != "" {
					fmt.Printf("Заметки:     %s\n", notes)
				}
			}

		case record.RecTypeText:
			fmt.Println("=== Текст заметки ===")
			if dataMap, ok := decryptedData.(map[string]interface{}); ok {
				if content, ok := dataMap["content"].(string); ok {
					fmt.Println(content)
				}
			}

		case record.RecTypeCard:
			fmt.Println("=== Данные карты ===")
			if dataMap, ok := decryptedData.(map[string]interface{}); ok {
				if cardNumber, ok := dataMap["card_number"].(string); ok {
					if showPassword {
						fmt.Printf("Номер карты: %s\n", cardNumber)
					} else {
						fmt.Printf("Номер карты: ****%s\n", cardNumber[len(cardNumber)-4:])
					}
				}
				if cardHolder, ok := dataMap["card_holder"].(string); ok {
					fmt.Printf("Держатель:   %s\n", cardHolder)
				}
				if showPassword {
					if cvv, ok := dataMap["cvv"].(string); ok {
						fmt.Printf("CVV:         %s\n", cvv)
					}
				}
			}

		case record.RecTypeBinary:
			fmt.Println("=== Файл ===")
			fmt.Printf("Размер:      %d байт\n", len(rec.EncryptedData))
			fmt.Println("Используйте команду 'export' для сохранения файла")
		}
	} else {
		// Данные не расшифрованы
		switch rec.Type {
		case record.RecTypeLogin:
			fmt.Println("=== Данные авторизации ===")
			fmt.Println("(Данные зашифрованы)")
			fmt.Println("Используйте --decrypt для расшифровки")

		case record.RecTypeText:
			fmt.Println("=== Текст заметки ===")
			fmt.Println("(Данные зашифрованы)")
			fmt.Println("Используйте --decrypt для расшифровки")

		case record.RecTypeCard:
			fmt.Println("=== Данные карты ===")
			fmt.Println("(Данные зашифрованы)")
			fmt.Println("Используйте --decrypt для расшифровки")

		case record.RecTypeBinary:
			fmt.Println("=== Файл ===")
			fmt.Printf("Размер:      %d байт\n", len(rec.EncryptedData))
			fmt.Println("Используйте команду 'export' для сохранения файла")
		}
	}

	return nil
}

func printRecordJSON(rec *client.LocalRecord, decryptedData interface{}, _ bool) error {
	output := struct {
		ID            int             `json:"id"`
		ServerID      int             `json:"server_id"`
		Type          string          `json:"type"`
		Meta          json.RawMessage `json:"meta"`
		Version       int             `json:"version"`
		LastModified  string          `json:"last_modified"`
		CreatedAt     string          `json:"created_at"`
		Synced        bool            `json:"synced"`
		DecryptedData interface{}     `json:"decrypted_data,omitempty"`
	}{
		ID:            rec.ID,
		ServerID:      rec.ServerID,
		Type:          string(rec.Type),
		Meta:          rec.Meta,
		Version:       rec.Version,
		LastModified:  rec.LastModified.Format(time.RFC3339),
		CreatedAt:     rec.CreatedAt.Format(time.RFC3339),
		Synced:        rec.Synced,
		DecryptedData: decryptedData,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func printRecordYAML(rec *client.LocalRecord, decryptedData interface{}, showPassword bool) error {
	return printRecordJSON(rec, decryptedData, showPassword)
}

func init() {
	GetCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "формат вывода (text, json, yaml)")
	GetCmd.Flags().BoolVar(&showPassword, "show-password", false, "показывать пароли и чувствительные данные")
	GetCmd.Flags().BoolVar(&decrypt, "decrypt", false, "расшифровать данные записи")
}
