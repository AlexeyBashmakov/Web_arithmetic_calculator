#!/bin/bash
# id = 3
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 3, \"expression\": \"3+4\"}" \
     -O -
# id = 1
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 1, \"expression\": \"1-1*3+3*9\"}" \
	 -O -
# id = 2, error in expression, for example
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 2, \"expression\": \"1-1*3+3*\"}" \
	 -O -
# id = 4
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 4, \"expression\": \"2.5*2\"}" \
	 -O -
# id = 5
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 5, \"expression\": \"3 / 2\"}" \
	 -O -
# id = 6
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 6, \"expression\": \" 1 - 3 + 9 \"}" \
	 -O -
# id = 7, error in answer, for example
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 7, \"expression\": \"(8+2*5)/(1+3*2-4)\"}" \
	 -O -
# id = 8
wget http://localhost:8080/api/v1/calculate \
	 --header="Content-Type: application/json" \
	 --post-data="{\"id\": 8, \"expression\": \"1+2*(3+4)-5\"}" \
	 -O -
