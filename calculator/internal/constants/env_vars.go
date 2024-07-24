package constants

const TimeAdd = "TIME_ADDITION_MS"         // время выполнения операции сложения в милисекундах
const TimeSub = "TIME_SUBSTRACTION_MS"     // время выполнения операции вычитания в милисекундах
const TimeMult = "TIME_MULTIPLICATIONS_MS" // время выполнения операции умножения в милисекундах
const TimeDiv = "TIME_DIVISIONS_MS"        // время выполнения операции деления в милисекундах

const TimeRequ = "TIME_REQUEST_FROM_AGENT" // время, через которое агент запрашивает у сервера задачу

const TimeTokenExpired = "TIME_EXPIRED_JWT_TOKEN" // время, через которое истекает время жизни JWT-токена пользователя

const CompPow = "COMPUTING_POWER" // количество горутин

const HTTP_PORT = "HTTP_PORT" // порт, на котором веб-сервис принимает запросы
const RPC_PORT = "RPC_PORT"   // порт, на котором по gRPC общаются сервер и агент
