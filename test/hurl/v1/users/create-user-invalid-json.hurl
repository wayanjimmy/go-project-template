# Test: Create user with invalid payload type returns bad request
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": 123,
  "email": "bad@example.com",
  "address": "Nowhere"
}

HTTP *
[Asserts]
status == 400
