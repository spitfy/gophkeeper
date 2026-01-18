// cmd/client/cmd/record/get.go
package record

import (
	"encoding/json"
	"fmt"
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
)

var GetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Просмотреть запись",
	Long: `Просмотр содержимого записи по ID.
	
Вы можете указать формат вывода и решить, показывать ли чувствительные данные.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
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

		switch outputFormat {
		case "json":
			return printRecordJSON(rec, showPassword)
		case "yaml":
			return printRecordYAML(rec, showPassword)
		default:
			return printRecordHuman(rec, showPassword)
		}
	},
}

func printRecordHuman(rec *client.LocalRecord, showPassword bool) error {
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

	switch rec.Type {
	case record.RecTypeLogin:
		fmt.Println("=== Данные авторизации ===")
		fmt.Println("(Данные зашифрованы)")
		if showPassword {
			fmt.Println("Используйте --decrypt для расшифровки")
		}

	case record.RecTypeText:
		fmt.Println("=== Текст заметки ===")
		fmt.Println("(Данные зашифрованы)")

	case record.RecTypeCard:
		fmt.Println("=== Данные карты ===")
		fmt.Println("(Данные зашифрованы)")

	case record.RecTypeBinary:
		fmt.Println("=== Файл ===")
		fmt.Printf("Размер:      %d байт\n", len(rec.EncryptedData))
		fmt.Println("Используйте команду 'export' для сохранения файла")
	}

	return nil
}

func printRecordJSON(rec *client.LocalRecord, showPassword bool) error {
	output := struct {
		ID           int             `json:"id"`
		ServerID     int             `json:"server_id"`
		Type         string          `json:"type"`
		Meta         json.RawMessage `json:"meta"`
		Version      int             `json:"version"`
		LastModified string          `json:"last_modified"`
		CreatedAt    string          `json:"created_at"`
		Synced       bool            `json:"synced"`
	}{
		ID:           rec.ID,
		ServerID:     rec.ServerID,
		Type:         string(rec.Type),
		Meta:         rec.Meta,
		Version:      rec.Version,
		LastModified: rec.LastModified.Format(time.RFC3339),
		CreatedAt:    rec.CreatedAt.Format(time.RFC3339),
		Synced:       rec.Synced,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func printRecordYAML(rec *client.LocalRecord, showPassword bool) error {
	return printRecordJSON(rec, showPassword)
}

func init() {
	GetCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "формат вывода (text, json, yaml)")
	GetCmd.Flags().BoolVar(&showPassword, "show-password", false, "показывать пароли и чувствительные данные")
}
