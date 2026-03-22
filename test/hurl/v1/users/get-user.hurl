# Test: Get existing user
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": "Hurl {{run_id}} Get",
  "email": "hurl-{{run_id}}-get@example.com",
  "address": "Street {{run_id}}"
}

HTTP *
[Captures]
user_id: jsonpath "$.id"
[Asserts]
status == 201
jsonpath "$.id" exists

GET {{base_url}}/v1/users/{{user_id}}

HTTP *
[Asserts]
status == 200
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{user_id}}"
jsonpath "$.name" == "Hurl {{run_id}} Get"
jsonpath "$.email" == "hurl-{{run_id}}-get@example.com"
jsonpath "$.address" == "Street {{run_id}}"
