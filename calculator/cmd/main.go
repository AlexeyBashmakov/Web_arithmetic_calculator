package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"web_arithmetic_calculator/http/server"
	"web_arithmetic_calculator/internal/agent"
	"web_arithmetic_calculator/internal/environ_vars"
)

func main() {
	if !environ_vars.CheckEnvironmentVariables() {
		if environ_vars.SetEnvironmentVariables() != nil {
			os.Exit(4)
		}
	}

	// настраиваем журналирование
	logger, logFile, err := setupLogger()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}
	// закроем журнальный файл
	defer logFile.Close()

	// будем прослушивать канал для приёма сигнала прерывания (например комбинации CTRL+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// запускаем сервер и получаем функцию, которая
	// элегантно завершает работу сервера без прерывания любых активных соединений (https://pkg.go.dev/net/http@go1.22.4#Server.Shutdown)
	ShutdownFunc, err := server.Run(ctx, logger)
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(2)
	}

	// https://tproger.ru/translations/go-web-server?ysclid=lwqqie5cje972900801
	// После получения сигнала выделим несколько секунд на корректное завершение работы сервера.
	// Попытка корректного завершения
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	_, cancel := context.WithCancel(context.Background())

	// создадим канал для сигнала агенту о завершении работы
	d := make(chan bool)
	// запуск агента - теперь он должен работать по gRPC
	if err = agent.Run(d, logger); err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}

	<-c
	go func() {
		// отправляем агенту сигнал о завершении работы
		d <- true
	}()
	cancel()
	// завершим работу сервера
	ShutdownFunc(ctx)

	os.Exit(0)
}

// настраиваем журналирование
func setupLogger() (*slog.Logger, *os.File, error) {
	// создаем журнальный файл для добавления записей
	f, err := os.OpenFile("calculator.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}
	// создаем текстовый handler для записи сообщений в файл
	h := slog.NewTextHandler(f, &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo})
	// создаем сам объект управляющий журналированием
	logger := slog.New(h)

	return logger, f, nil
}

/* веб-сервис по расчёту арифметических выражений
для расчёта каждого выражения оное преобразуется в обратную польскую нотацию,
которая позволяет производить расчёты не связанных действий параллельно
*/
