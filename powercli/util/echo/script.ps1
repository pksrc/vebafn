Write-Output $args 

$json = $args | ConvertFrom-Json

if("2F232EB71D584140B9529460340FCFE4" -eq $json.key) {
    Invoke-WebRequest $json.response_url -Method 'POST' -Headers @{'Content-Type' = 'application/json; charset=utf-8'} -Body "{'text': '${json}','response_type': 'ephemeral'}"
} else {
    Write-Output "Unable to Verify the Key from Gateway"
} 