# Test: Search users returns empty result for unknown query
GET {{base_url}}/v1/search/users?q=no-match-{{run_id}}&limit=5

HTTP *
[Asserts]
status == 200
header "Content-Type" contains "application/json"
jsonpath "$" count == 0
