# Test: Get non-existing user returns not found
GET {{base_url}}/v1/users/00000000-0000-0000-0000-000000000000

HTTP *
[Asserts]
status == 404
header "Content-Type" contains "application/json"
jsonpath "$.error.code" == "not_found"
jsonpath "$.error.message" == "user not found"
jsonpath "$.error.details.resource" == "user"
jsonpath "$.error.details.id" == "00000000-0000-0000-0000-000000000000"
