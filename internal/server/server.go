package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fpawel/signature-service/internal/api/swagger/generated/models"
	"github.com/fpawel/signature-service/internal/api/swagger/generated/restapi"
	"github.com/fpawel/signature-service/internal/api/swagger/generated/restapi/operations"
	"github.com/fpawel/signature-service/internal/httph"
	"github.com/fpawel/signature-service/internal/httph/jsoniter"
	"github.com/fpawel/signature-service/internal/httph/log_http"
	"github.com/fpawel/signature-service/internal/httph/requestid"
	"github.com/fpawel/signature-service/internal/repository/inmem"
	"github.com/fpawel/signature-service/internal/service"
	"github.com/fpawel/signature-service/internal/signing"
	"github.com/go-openapi/loads"
)

const (
	serviceName      = "Signature Service"
	headerServerName = "X-Server"
)

// Run - основная функция запуска сервера
func Run(port int) error {
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		return err
	}

	serviceApi := operations.NewSignatureServiceAPI(swaggerSpec)

	signReg := signing.NewRegistry()
	signReg.RegisterSigner(models.CreateSignatureDeviceRequestAlgorithmECC, signing.ECDSASigner{})
	signReg.RegisterSigner(models.CreateSignatureDeviceRequestAlgorithmRSA, signing.RSASigner{})

	serv := service.NewService(inmem.NewStorage(), signReg)

	serviceApi.SignTransactionHandler = operations.SignTransactionHandlerFunc(serv.SignTransaction)
	serviceApi.CreateSignatureDeviceHandler = operations.CreateSignatureDeviceHandlerFunc(serv.CreateSignatureDevice)
	serviceApi.GetSignatureDeviceHandler = operations.GetSignatureDeviceHandlerFunc(serv.GetSignatureDevice)
	serviceApi.ListSignatureDevicesHandler = operations.ListSignatureDevicesHandlerFunc(serv.ListSignatureDevices)
	serviceApi.ListDeviceSignaturesHandler = operations.ListDeviceSignaturesHandlerFunc(serv.ListDeviceSignatures)

	// создать новый сервер с предоставленным API
	server := restapi.NewServer(serviceApi)
	// Установить порт, на котором будет работать сервер
	server.Port = port
	// сразу регистрируем остановку сервера по завершении App.Run().
	defer func() {
		if err = server.Shutdown(); err != nil {
			slog.Error("Error when shutdown", "error", err)
		}
	}()

	// Установка промежуточного ПО, вызываемые в Апи.
	// ОБЯЗАТЕЛЬНО выполнить ДО server.ConfigureAPI().
	restapi.SetupApiMiddlewares = httph.Use(
		log_http.Handler,
		httph.Recovery(false),
	)
	// Установка промежуточного ПО, вызываемые как в Апи, так и во всех внешних точках, таких как /metrics.
	// ОБЯЗАТЕЛЬНО выполнить ДО server.ConfigureAPI()
	restapi.SetupGlobalMiddleware = httph.Use(
		httph.SetResponseHeader(headerServerName, serviceName), // установить в ответах заголовок с именем сервиса
		requestid.Handler, // добавляет в контекст запроса цифровой отпечаток
	)
	//------------------------------------------------------------------------------------------------------------------
	// Настройки API по умолчанию.
	// ОБЯЗАТЕЛЬНО выполнить ПОСЛЕ установки промежуточного ПО и ДО общей настройки API.
	server.ConfigureAPI()
	//------------------------------------------------------------------------------------------------------------------
	// Общая настройка API.
	// ОБЯЗАТЕЛЬНО выполнить ПОСЛЕ server.ConfigureAPI()
	serviceApi.Logger = func(format string, args ...interface{}) {
		slog.Info("⚔️ " + fmt.Sprintf(format, args...))
	}
	serviceApi.JSONConsumer = jsoniter.JSONConsumer() // Указываем потребителя для MIME Type application/json
	serviceApi.JSONProducer = jsoniter.JSONProducer() // Указываем производителя для MIME Type application/json

	// Сервер будет обрабатывать сигнал прерывания, поэтому мы можем зарегистрировать для него обработчики.
	serviceApi.PreServerShutdown = func() {
		slog.Warn("⚔️ Сервер прекращает свою работу...")
	}
	serviceApi.ServerShutdown = func() {
		slog.Warn("⚔️ Сервер завершил свою работу")
	}
	//------------------------------------------------------------------------------------------------------------------
	if err = server.Serve(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}
