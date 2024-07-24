#!/bin/bash

export COMPUTING_POWER=2
# переменные времени ниже задаются в миллисекундах
export TIME_ADDITION_MS=2000
export TIME_SUBSTRACTION_MS=2000
export TIME_MULTIPLICATIONS_MS=2000
export TIME_DIVISIONS_MS=2000

export TIME_REQUEST_FROM_AGENT=2000

# переменные времени ниже задаются в минутах
export TIME_EXPIRED_JWT_TOKEN=120

export HTTP_PORT=8080
export RPC_PORT=5001

./main
