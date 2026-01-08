// cmd/client/cmd/record/list.go
package record

import (
	"encoding/json"
	"fmt"
	"gophkeeper/internal/app/client"
	"gophkeeper/internal/domain/record"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var (
	listType    string
	listFormat  string
	showDeleted bool
	limit       int
	offset      int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Список записей",
	Long: `Просмотр списка всех записей с возможностью фильтрации по типу.
	
Поддерживается пагинация через флаги --limit и --offset.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		// Получаем записи
		records, err := app.ListRecords(cmd.Context(), record.ListRequest{
			Type:        record.Type(listType),
			ShowDeleted: showDeleted,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return fmt.Errorf("ошибка получения списка записей: %w", err)
		}

		// Выводим результат
		switch listFormat {
		case "json":
			return printRecordsJSON(records)
		case "table":
			return printRecordsTable(records)
		case "csv":
			return printRecordsCSV(records)
		default:
			return printRecordsSimple(records)
		}
	},
}

func printRecordsSimple(records []*record.Record) error {
	if len(records) == 0 {
		fmt.Println("Записи не найдены")
		return nil
	}

	fmt.Printf("Найдено записей: %d\n\n", len(records))

	for i, rec := range records {
		status := "✓"
		if rec.Deleted {
			status = "✗"
		}

		fmt.Printf("%d. [%s] %s (%s)\n", i+1, status, rec.Metadata.Name, rec.Type)
		if rec.Metadata.Description != "" {
			fmt.Printf("   %s\n", rec.Metadata.Description)
		}
		fmt.Printf("   ID: %s | Создано: %s\n",
			rec.ID,
			rec.CreatedAt.Format("2006-01-02"))
		fmt.Println()
	}

	return nil
}

func printRecordsTable(records []*record.Record) error {
	if len(records) == 0 {
		fmt.Println("Записи не найдены")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tТип\tНазвание\tСтатус\tСоздано\tОбновлено\t\n")
	fmt.Fprintf(w, "---\t---\t---\t---\t---\t---\t\n")

	for _, rec := range records {
		status := "Активна"
		if rec.Deleted {
			status = "Удалена"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
			shortID(rec.ID, 8),
			string(rec.Type),
			truncate(rec.Metadata.Name, 30),
			status,
			rec.CreatedAt.Format("2006-01-02"),
			rec.UpdatedAt.Format("2006-01-02"),
		)
	}

	w.Flush()
	fmt.Printf("\nВсего записей: %d\n", len(records))
	return nil
}

func printRecordsJSON(records []*record.Record) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

func printRecordsCSV(records []*record.Record) error {
	fmt.Println("ID,Type,Name,Description,Status,CreatedAt,UpdatedAt")

	for _, rec := range records {
		status := "active"
		if rec.Deleted {
			status = "deleted"
		}

		fmt.Printf("%s,%s,%q,%q,%s,%s,%s\n",
			rec.ID,
			string(rec.Type),
			rec.Metadata.Name,
			rec.Metadata.Description,
			status,
			rec.CreatedAt.Format(time.RFC3339),
			rec.UpdatedAt.Format(time.RFC3339),
		)
	}

	return nil
}

func shortID(id string, length int) string {
	if len(id) <= length {
		return id
	}
	return id[:length] + "..."
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func init() {
	listCmd.Flags().StringVarP(&listType, "type", "t", "", "фильтр по типу записи")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "simple", "формат вывода (simple, table, json, csv)")
	listCmd.Flags().BoolVar(&showDeleted, "deleted", false, "показывать удаленные записи")
	listCmd.Flags().IntVar(&limit, "limit", 50, "ограничение количества записей")
	listCmd.Flags().IntVar(&offset, "offset", 0, "смещение для пагинации")
}
