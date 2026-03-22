# Test: Delete existing user
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": "Hurl {{run_id}} Delete",
  "email": "hurl-{{run_id}}-delete@example.com",
  "address": "Delete {{run_id}}"
}

HTTP *
[Captures]
user_id: jsonpath "$.id"
[Asserts]
status == 201
jsonpath "$.id" exists

DELETE {{base_url}}/v1/users/{{user_id}}

HTTP *
[Asserts]
status == 204

GET {{base_url}}/v1/users/{{user_id}}

HTTP *
[Asserts]
status == 404
body contains "not found"
