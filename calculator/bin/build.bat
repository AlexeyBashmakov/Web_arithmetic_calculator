go mod tidy

if not defined CGO_ENABLED (
  ::echo "CGO_ENABLED не установлена"
  set CGO_ENABLED=1
) else (
  echo CGO_ENABLED=%CGO_ENABLED%
)

go build -ldflags="-s -w" ..\cmd\main.go
