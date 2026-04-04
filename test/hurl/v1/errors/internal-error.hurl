# Test: Internal error example returns structured internal server error
GET {{base_url}}/v1/examples/errors/internal

HTTP *
[Asserts]
status == 500
header "Content-Type" contains "application/json"
jsonpath "$.error.code" == "internal"
jsonpath "$.error.message" == "internal server error"
