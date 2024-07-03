package to_log

import (
	"fmt"
	"log/slog"
)

// структура для упаковывания параметров функции prnt
type Param struct {
	Msg    string
	Logger *slog.Logger
	Level  int
}

// функция для вывода сообщения в консоль и журнальный файл
// func prnt(msg string, logger *slog.Logger) {
func Print(prm Param) {
	fmt.Println(prm.Msg)
	if prm.Level == 0 {
		prm.Logger.Info(prm.Msg)
	} else {
		prm.Logger.Warn(prm.Msg)
	}
}
