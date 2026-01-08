package record

import (
	"github.com/spf13/cobra"
)

// RecordCmd - родительская команда для всех операций с записями
var RecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Управление записями",
	Long:  `Создание, просмотр, обновление и удаление защищенных записей.`,
}
