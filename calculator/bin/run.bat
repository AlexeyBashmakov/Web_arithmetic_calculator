@echo off

:: <- так начинается комментарий

set COMPUTING_POWER=2
:: переменные времени ниже задаются в миллисекундах
set TIME_ADDITION_MS=2000
set TIME_SUBSTRACTION_MS=2000
set TIME_MULTIPLICATIONS_MS=2000
set TIME_DIVISIONS_MS=2000

set TIME_REQUEST_FROM_AGENT=2000

:: переменные времени ниже задаются в минутах
set TIME_EXPIRED_JWT_TOKEN=120

set HTTP_PORT=8080
set RPC_PORT=5000

main.exe & call;

