# Test: Get non-existing user returns not found
GET {{base_url}}/v1/users/00000000-0000-0000-0000-000000000000

HTTP *
[Asserts]
status == 404
body contains "not found"
