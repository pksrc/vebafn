# Fetch the VC Credentials
    $VC_CONFIG_FILE = "/var/openfaas/secrets/vcconfigjson"

    $VC_CONFIG = (Get-Content -Raw -Path $VC_CONFIG_FILE | ConvertFrom-Json)
    if($env:function_debug -eq "true") {
        Write-host "DEBUG: `"$VC_CONFIG`""
    }

    Set-PowerCLIConfiguration -InvalidCertificateAction Ignore  -DisplayDeprecationWarnings $false -ParticipateInCeip $false -Confirm:$false
write-host "write your powercli code here"
