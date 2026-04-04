# Test: Create user validation error returns structured bad request
POST {{base_url}}/v1/users
Content-Type: application/json
{
  "name": "Hurl validation example"
}

HTTP *
[Asserts]
status == 400
header "Content-Type" contains "application/json"
jsonpath "$.error.code" == "invalid_argument"
jsonpath "$.error.message" == "request validation failed"
jsonpath "$.error.details[?(@.field=='email')]" count == 1
jsonpath "$.error.details[?(@.field=='address')]" count == 1
