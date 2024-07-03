rem два следующих синтаксиса не работают при перечаде json в винде
rem curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{"id": 1, "expression": "1+1"}" -v
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{'id': 1, 'expression': '1+1'}" -v

::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 1, \"expression\": \"1-1\"}" -v

echo "Adding an arithpmetic expression for calculate"
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 1, \"expression\": \"1-1*3+3*9\"}" -v

::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 2, \"expression\": \"1-1*3+3*\"}" -v

::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 3, \"expression\": \"3+4\"}" -v
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 4, \"expression\": \"2.5*2\"}" -v
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 5, \"expression\": \"3 / 2\"}" -v
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 6, \"expression\": \" 1 - 3 + 9 \"}" -v
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 7, \"expression\": \"(8+2*5)/(1+3*2-4)\"}" -v
::curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"id\": 8, \"expression\": \"1+2*(3+4)-5\"}" -v

curl --location "http://localhost:8080/api/v1/calculate" --header "Content-Type: application/json" --data "{\"expression\": \"3+4\"}" -v
