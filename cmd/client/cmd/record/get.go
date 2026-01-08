// cmd/client/cmd/record/get.go
package record

import (
	"encoding/json"
	"fmt"
	"gophkeeper/internal/app/client"
	"gophkeeper/internal/domain/record"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	showPassword bool
)

var getCmd = &cobra.Command{
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

		recordID := args[0]

		// Получаем запись
		rec, err := app.GetRecord(cmd.Context(), recordID)
		if err != nil {
			return fmt.Errorf("ошибка получения записи: %w", err)
		}

		// Выводим в зависимости от формата
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

func printRecordHuman(rec *record.Record, showPassword bool) error {
	fmt.Printf("ID:          %s\n", rec.ID)
	fmt.Printf("Тип:         %s\n", rec.Type)
	fmt.Printf("Название:    %s\n", rec.Metadata.Name)

	if rec.Metadata.Description != "" {
		fmt.Printf("Описание:    %s\n", rec.Metadata.Description)
	}

	fmt.Printf("Создано:     %s\n", rec.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Обновлено:   %s\n", rec.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Версия:      %d\n", rec.Version)
	fmt.Println()

	// Выводим данные в зависимости от типа
	switch rec.Type {
	case record.TypePassword:
		var data map[string]string
		if err := json.Unmarshal(rec.Data, &data); err != nil {
			return err
		}

		fmt.Println("=== Данные авторизации ===")
		fmt.Printf("Логин:       %s\n", data["username"])
		if showPassword {
			fmt.Printf("Пароль:      %s\n", data["password"])
		} else {
			fmt.Printf("Пароль:      ********\n")
		}
		if data["url"] != "" {
			fmt.Printf("URL:         %s\n", data["url"])
		}

	case record.TypeText:
		fmt.Println("=== Текст заметки ===")
		fmt.Println(string(rec.Data))

	case record.TypeCard:
		var data map[string]string
		if err := json.Unmarshal(rec.Data, &data); err != nil {
			return err
		}

		fmt.Println("=== Данные карты ===")
		fmt.Printf("Номер:       %s\n", maskCardNumber(data["number"]))
		fmt.Printf("Держатель:   %s\n", data["holder"])
		fmt.Printf("Срок:        %s\n", data["expiryDate"])
		if showPassword {
			fmt.Printf("CVV:         %s\n", data["cvv"])
		} else {
			fmt.Printf("CVV:         ***\n")
		}

	case record.TypeBinary:
		fmt.Println("=== Файл ===")
		fmt.Printf("Размер:      %d байт\n", len(rec.Data))
		fmt.Printf("Имя файла:   %s\n", rec.Metadata.Description)
		fmt.Println("Используйте команду 'export' для сохранения файла")
	}

	return nil
}

func printRecordJSON(rec *record.Record, showPassword bool) error {
	output := struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Metadata  record.Metadata `json:"metadata"`
		Data      interface{}     `json:"data"`
		CreatedAt string          `json:"created_at"`
		UpdatedAt string          `json:"updated_at"`
		Version   int             `json:"version"`
	}{
		ID:        rec.ID,
		Type:      string(rec.Type),
		Metadata:  rec.Metadata,
		CreatedAt: rec.CreatedAt.Format(time.RFC3339),
		UpdatedAt: rec.UpdatedAt.Format(time.RFC3339),
		Version:   rec.Version,
	}

	// Обрабатываем данные в зависимости от типа
	switch rec.Type {
	case record.TypePassword, record.TypeCard:
		var data map[string]string
		if err := json.Unmarshal(rec.Data, &data); err != nil {
			return err
		}

		if !showPassword {
			if rec.Type == record.TypePassword {
				data["password"] = "********"
			} else if rec.Type == record.TypeCard {
				data["cvv"] = "***"
				data["number"] = maskCardNumber(data["number"])
			}
		}
		output.Data = data

	default:
		output.Data = string(rec.Data)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func maskCardNumber(number string) string {
	if len(number) < 4 {
		return number
	}
	return "**** **** **** " + number[len(number)-4:]
}

func printRecordYAML(rec *record.Record, showPassword bool) error {
	// Для простоты используем JSON с преобразованием в YAML
	// В реальном проекте можно использовать go-yaml
	return printRecordJSON(rec, showPassword)
}

func init() {
	getCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "формат вывода (text, json, yaml)")
	getCmd.Flags().BoolVar(&showPassword, "show-password", false, "показывать пароли и чувствительные данные")
}
