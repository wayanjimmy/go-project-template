# Test: Create user
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": "Hurl {{run_id}} Create",
  "email": "hurl-{{run_id}}-create@example.com",
  "address": "Address {{run_id}}"
}

HTTP *
[Captures]
user_id: jsonpath "$.id"
[Asserts]
status == 201
header "Content-Type" contains "application/json"
jsonpath "$.id" exists
jsonpath "$.name" == "Hurl {{run_id}} Create"
jsonpath "$.email" == "hurl-{{run_id}}-create@example.com"
jsonpath "$.address" == "Address {{run_id}}"
