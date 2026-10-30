# Test: Search users returns created user
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": "Hurl Search {{run_id}}",
  "email": "hurl-search-{{run_id}}@example.com",
  "address": "Search Address {{run_id}} {{run_token}}"
}

HTTP *
[Captures]
user_id: jsonpath "$.id"
[Asserts]
status == 201
jsonpath "$.id" exists

GET {{base_url}}/v1/search/users?q={{run_id}}&limit=20

HTTP *
[Asserts]
status == 200

GET {{base_url}}/v1/search/users?q={{run_id}}&limit=20

HTTP *
[Asserts]
status == 200

GET {{base_url}}/v1/search/users?q={{run_id}}&limit=20

HTTP *
[Asserts]
status == 200
header "Content-Type" contains "application/json"
jsonpath "$" count == 1
jsonpath "$[0].id" == "{{user_id}}"
jsonpath "$[0].name" == "Hurl Search {{run_id}}"
jsonpath "$[0].email" == "hurl-search-{{run_id}}@example.com"
