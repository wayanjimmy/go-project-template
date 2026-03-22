# Test: Update existing user
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": "Hurl {{run_id}} Before",
  "email": "hurl-{{run_id}}-before@example.com",
  "address": "Before {{run_id}}"
}

HTTP *
[Captures]
user_id: jsonpath "$.id"
[Asserts]
status == 201
jsonpath "$.id" exists

PUT {{base_url}}/v1/users/{{user_id}}
Content-Type: application/json
{
  "name": "Hurl {{run_id}} After",
  "email": "hurl-{{run_id}}-after@example.com",
  "address": "After {{run_id}}"
}

HTTP *
[Asserts]
status == 200
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{user_id}}"
jsonpath "$.name" == "Hurl {{run_id}} After"
jsonpath "$.email" == "hurl-{{run_id}}-after@example.com"
jsonpath "$.address" == "After {{run_id}}"

GET {{base_url}}/v1/users/{{user_id}}

HTTP *
[Asserts]
status == 200
jsonpath "$.id" == "{{user_id}}"
jsonpath "$.name" == "Hurl {{run_id}} After"
jsonpath "$.email" == "hurl-{{run_id}}-after@example.com"
jsonpath "$.address" == "After {{run_id}}"
