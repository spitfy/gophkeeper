package record

import (
	"context"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"gophkeeper/internal/app/server/api/http/middleware/auth"
	"gophkeeper/internal/domain/record"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/exp/slog"
)

type Handler struct {
	service    record.Servicer
	log        *slog.Logger
	middleware huma.Middlewares
}

func NewHandler(service record.Servicer, log *slog.Logger, mws huma.Middlewares) *Handler {
	return &Handler{
		service:    service,
		log:        log,
		middleware: mws,
	}
}

func (h *Handler) SetupRoutes(api huma.API) {
	// Generic CRUD
	huma.Register(api, h.listOp(), h.list)
	huma.Register(api, h.createOp(), h.create)
	huma.Register(api, h.findOp(), h.find)
	huma.Register(api, h.updateOp(), h.update)
	huma.Register(api, h.deleteOp(), h.delete)

	// Typed create handlers
	huma.Register(api, h.createLoginOp(), h.createLogin)
	huma.Register(api, h.createTextOp(), h.createText)
	huma.Register(api, h.createCardOp(), h.createCard)
	huma.Register(api, h.createBinaryOp(), h.createBinary)
}

func (h *Handler) list(ctx context.Context, _ *struct{}) (*listOutput, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	records, err := h.service.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &listOutput{
		Body: records,
	}, nil
}

func (h *Handler) find(ctx context.Context, input *findInput) (*findOutput, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	rec, err := h.service.Find(ctx, userID, input.ID)
	if err != nil {
		return &findOutput{
			Body: findResponse{
				Status: "Error",
			},
		}, err
	}

	return &findOutput{
		Body: findResponse{
			Status: "Ok",
			Record: rec,
		},
	}, nil
}

func (h *Handler) create(ctx context.Context, input *createInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	recordID, err := h.service.Create(ctx, userID, input.Body.Type, input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &output{
			Body: response{Status: "Error"},
		}, err
	}

	return &output{
		Body: response{
			ID:     recordID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) update(ctx context.Context, input *updateInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	err := h.service.Update(ctx, userID, input.ID, input.Body.Type, input.Body.EncryptedData, input.Body.Meta)
	if err != nil {
		return &output{
			Body: response{
				ID:     input.ID,
				Status: "Error",
			},
		}, err
	}
	return &output{
		Body: response{
			ID:     input.ID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) delete(ctx context.Context, input *updateInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	err := h.service.Delete(ctx, userID, input.ID)
	if err != nil {
		return &output{
			Body: response{
				Status: "Error",
			},
		}, err
	}
	return &output{
		Body: response{
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) createLogin(ctx context.Context, input *createLoginInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	// Создаем данные логина
	loginData := &record.LoginData{
		Username: input.Body.Username,
		Password: input.Body.Password,
		Notes:    input.Body.Notes,
	}

	// Создаем метаданные
	loginMeta := &record.LoginMeta{
		Title:     input.Body.Title,
		Resource:  input.Body.Resource,
		Category:  input.Body.Category,
		Tags:      input.Body.Tags,
		TwoFA:     input.Body.TwoFA,
		TwoFAType: input.Body.TwoFAType,
	}

	// Валидация
	if err := loginData.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	if err := loginMeta.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	// Создание записи
	recordID, err := h.service.CreateWithModels(ctx, userID,
		record.RecTypeLogin,
		loginData,
		loginMeta,
		input.Body.DeviceID,
	)

	if err != nil {
		return &output{
			Body: response{Status: "Error", Message: err.Error()},
		}, err
	}

	return &output{
		Body: response{
			ID:     recordID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) createText(ctx context.Context, input *createTextInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	// Создаем данные текста
	textData := &record.TextData{
		Content: input.Body.Content,
	}

	// Создаем метаданные
	textMeta := &record.TextMeta{
		Title:       input.Body.Title,
		Category:    input.Body.Category,
		Tags:        input.Body.Tags,
		Format:      input.Body.Format,
		Language:    input.Body.Language,
		IsSensitive: input.Body.IsSensitive,
		WordCount:   countWords(input.Body.Content),
		CharsCount:  utf8.RuneCountInString(input.Body.Content),
		Preview:     truncateString(input.Body.Content, 100),
	}

	// Валидация
	if err := textData.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	if err := textMeta.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	// Создание записи
	recordID, err := h.service.CreateWithModels(ctx, userID,
		record.RecTypeText,
		textData,
		textMeta,
		input.Body.DeviceID,
	)

	if err != nil {
		return &output{
			Body: response{Status: "Error", Message: err.Error()},
		}, err
	}

	return &output{
		Body: response{
			ID:     recordID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) createCard(ctx context.Context, input *createCardInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	// Создаем данные карты
	cardData := &record.CardData{
		CardNumber:     input.Body.CardNumber,
		CardHolder:     input.Body.CardHolder,
		ExpiryMonth:    input.Body.ExpiryMonth,
		ExpiryYear:     input.Body.ExpiryYear,
		CVV:            input.Body.CVV,
		PIN:            input.Body.PIN,
		BillingAddress: input.Body.BillingAddress,
	}

	// Создаем метаданные
	cardMeta := &record.CardMeta{
		Title:         input.Body.Title,
		BankName:      input.Body.BankName,
		PaymentSystem: input.Body.PaymentSystem,
		Category:      input.Body.Category,
		Tags:          input.Body.Tags,
		Notes:         input.Body.Notes,
		IsVirtual:     input.Body.IsVirtual,
		IsActive:      input.Body.IsActive,
		DailyLimit:    input.Body.DailyLimit,
		PhoneNumber:   input.Body.PhoneNumber,
	}

	// Валидация
	if err := cardData.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	if err := cardMeta.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	// Создание записи
	recordID, err := h.service.CreateWithModels(ctx, userID,
		record.RecTypeCard,
		cardData,
		cardMeta,
		input.Body.DeviceID,
	)

	if err != nil {
		return &output{
			Body: response{Status: "Error", Message: err.Error()},
		}, err
	}

	return &output{
		Body: response{
			ID:     recordID,
			Status: "Ok",
		},
	}, nil
}

func (h *Handler) createBinary(ctx context.Context, input *createBinaryInput) (*output, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	// Декодируем base64 данные
	decodedData, err := base64.StdEncoding.DecodeString(input.Body.Data)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity("Invalid base64 data: " + err.Error())
	}

	// Создаем данные бинарного файла
	binaryData := &record.BinaryData{
		Data:        decodedData,
		Filename:    input.Body.Filename,
		ContentType: input.Body.ContentType,
		Size:        int64(len(decodedData)),
	}

	// Создаем метаданные
	binaryMeta := &record.BinaryMeta{
		Title:        input.Body.Title,
		Category:     input.Body.Category,
		Tags:         input.Body.Tags,
		Description:  input.Body.Description,
		OriginalHash: input.Body.OriginalHash,
		Dimensions:   input.Body.Dimensions,
		Duration:     input.Body.Duration,
	}

	// Добавляем информацию о сжатии и шифровании если предоставлены
	if input.Body.Compression != nil {
		binaryMeta.Compression = *input.Body.Compression
	}
	if input.Body.Encryption != nil {
		binaryMeta.Encryption = *input.Body.Encryption
	}

	// Валидация
	if err := binaryData.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	if err := binaryMeta.Validate(); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	// Создание записи
	recordID, err := h.service.CreateWithModels(ctx, userID,
		record.RecTypeBinary,
		binaryData,
		binaryMeta,
		input.Body.DeviceID,
	)

	if err != nil {
		return &output{
			Body: response{Status: "Error", Message: err.Error()},
		}, err
	}

	return &output{
		Body: response{
			ID:     recordID,
			Status: "Ok",
		},
	}, nil
}

// Helper functions

func countWords(s string) int {
	words := strings.Fields(s)
	return len(words)
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
