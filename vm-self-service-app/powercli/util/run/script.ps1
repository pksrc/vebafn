# Process payload received from the Gateway
$json = $args | ConvertFrom-Json
if($env:function_debug -eq "true") {
    Write-Host "DEBUG: json=`"$($json | Format-List | Out-String)`""
}

$keyword = "invoke"

$pos = $json.text.IndexOf($keyword)
$leftPart = $json.text.Substring(0, $pos+$keyword.Length)
Write-Host "l"+$leftPart

$rightPart = $json.text.Substring($pos+$keyword.Length+1)
Write-Host "r"+$rightPart

# Validate that the request is indeed from the Gateway function before allowing any critical functionality
if(("2F232EB71D584140B9529460340FCFE4" -eq $json.key) -and ("invoke" -eq $leftPart)) {
    # Fetch the VC Credentials
    $VC_CONFIG_FILE = "/var/openfaas/secrets/vcconfigjson"

    $VC_CONFIG = (Get-Content -Raw -Path $VC_CONFIG_FILE | ConvertFrom-Json)
    if($env:function_debug -eq "true") {
        Write-host "DEBUG: `"$VC_CONFIG`""
    }

    Set-PowerCLIConfiguration -Scope User -ParticipateInCEIP $false -InvalidCertificateAction Ignore  -DisplayDeprecationWarnings $false -Confirm:$false | Out-Null

    # Connect to vCenter Server
    Connect-VIServer -Server $($VC_CONFIG.VC) -User $($VC_CONFIG.VC_USERNAME) -Password $($VC_CONFIG.VC_PASSWORD)

    $rightPart = $rightPart -replace [System.Environment]::NewLine, ' '
    Write-Host "CMD"+$rightPart
    Invoke-Expression $rightPart | Tee-Object -Variable out 
    Write-Host "DEBUG: `"$out`""

    [string]$out = $out -replace ":", "=" -replace "{", "(" -replace "}", ")"
    Write-Host "MANIPULATED: `"$out`""
    Invoke-WebRequest $json.response_url -Method 'POST' -Headers @{'Content-Type' = 'application/json; charset=utf-8'} -Body "{'text': '$out','response_type': 'in_channel'}"

    Disconnect-VIServer * -Confirm:$false
} else {
    Write-Output "Unable to Verify the Key from Gateway"
}
