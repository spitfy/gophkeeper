// cmd/client/cmd/record/list.go
package record

import (
	"encoding/json"
	"fmt"
	"gophkeeper/internal/app/client"
	"gophkeeper/internal/domain/record"
	"os"
	"strconv"
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

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Список записей",
	Long: `Просмотр списка всех записей с возможностью фильтрации по типу.
	
Поддерживается пагинация через флаги --limit и --offset.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		filter := &client.RecordFilter{
			Type:        record.RecType(listType),
			ShowDeleted: showDeleted,
			Limit:       limit,
			Offset:      offset,
		}

		records, err := app.ListRecords(cmd.Context(), filter)
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

func printRecordsSimple(records []*client.LocalRecord) error {
	if len(records) == 0 {
		fmt.Println("Записи не найдены")
		return nil
	}

	fmt.Printf("Найдено записей: %d\n\n", len(records))

	for i, rec := range records {
		status := "✓"
		if rec.DeletedAt != nil {
			status = "✗"
		}

		var meta map[string]interface{}
		title := "Без названия"
		if len(rec.Meta) > 0 {
			if err := json.Unmarshal(rec.Meta, &meta); err == nil {
				if t, ok := meta["title"].(string); ok {
					title = t
				}
			}
		}

		fmt.Printf("%d. [%s] %s (%s)\n", i+1, status, title, rec.Type)
		fmt.Printf("   ID: %d | Server ID: %d | Создано: %s\n",
			rec.ID,
			rec.ServerID,
			rec.CreatedAt.Format("2006-01-02"))
		fmt.Println()
	}

	return nil
}

func printRecordsTable(records []*client.LocalRecord) error {
	if len(records) == 0 {
		fmt.Println("Записи не найдены")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tServer ID\tТип\tНазвание\tСтатус\tСоздано\tОбновлено\t\n")
	fmt.Fprintf(w, "---\t---\t---\t---\t---\t---\t---\t\n")

	for _, rec := range records {
		status := "Активна"
		if rec.DeletedAt != nil {
			status = "Удалена"
		}

		var meta map[string]interface{}
		title := "Без названия"
		if len(rec.Meta) > 0 {
			if err := json.Unmarshal(rec.Meta, &meta); err == nil {
				if t, ok := meta["title"].(string); ok {
					title = t
				}
			}
		}

		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\t%s\t\n",
			rec.ID,
			rec.ServerID,
			string(rec.Type),
			truncate(title, 30),
			status,
			rec.CreatedAt.Format("2006-01-02"),
			rec.LastModified.Format("2006-01-02"),
		)
	}

	w.Flush()
	fmt.Printf("\nВсего записей: %d\n", len(records))
	return nil
}

func printRecordsJSON(records []*client.LocalRecord) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

func printRecordsCSV(records []*client.LocalRecord) error {
	fmt.Println("ID,ServerID,Type,Title,Status,CreatedAt,UpdatedAt")

	for _, rec := range records {
		status := "active"
		if rec.DeletedAt != nil {
			status = "deleted"
		}

		var meta map[string]interface{}
		title := "Без названия"
		if len(rec.Meta) > 0 {
			if err := json.Unmarshal(rec.Meta, &meta); err == nil {
				if t, ok := meta["title"].(string); ok {
					title = t
				}
			}
		}

		fmt.Printf("%d,%d,%s,%q,%s,%s,%s\n",
			rec.ID,
			rec.ServerID,
			string(rec.Type),
			title,
			status,
			rec.CreatedAt.Format(time.RFC3339),
			rec.LastModified.Format(time.RFC3339),
		)
	}

	return nil
}

func shortID(id int) string {
	return strconv.Itoa(id)
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func init() {
	ListCmd.Flags().StringVarP(&listType, "type", "t", "", "фильтр по типу записи")
	ListCmd.Flags().StringVarP(&listFormat, "format", "f", "simple", "формат вывода (simple, table, json, csv)")
	ListCmd.Flags().BoolVar(&showDeleted, "deleted", false, "показывать удаленные записи")
	ListCmd.Flags().IntVar(&limit, "limit", 50, "ограничение количества записей")
	ListCmd.Flags().IntVar(&offset, "offset", 0, "смещение для пагинации")
}
