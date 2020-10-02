# Process payload received from the Gateway
$json = $args | ConvertFrom-Json
if($env:function_debug -eq "true") {
    Write-Host "DEBUG: json=`"$($json | Format-List | Out-String)`""
}
Write-Host "DEBUG: json=`"$($json | Format-List | Out-String)`""

$separator = [System.Environment]::NewLine
$option = [System.StringSplitOptions]::RemoveEmptyEntries
$inputArr = $json.text.split($separator,$option)

# Validate that the request is indeed from the Gateway function before allowing any critical functionality
if(("2F232EB71D584140B9529460340FCFE4" -eq $json.key) -and ("transform" -eq $inputArr[0].trim())) {
    # Fetch the VC Credentials
    $VC_CONFIG_FILE = "/var/openfaas/secrets/vcconfigjson"

    $VC_CONFIG = (Get-Content -Raw -Path $VC_CONFIG_FILE | ConvertFrom-Json)
    if($env:function_debug -eq "true") {
        Write-host "DEBUG: `"$VC_CONFIG`""
    }

    Set-PowerCLIConfiguration -Scope User -ParticipateInCEIP $false -InvalidCertificateAction Ignore  -DisplayDeprecationWarnings $false -Confirm:$false | Out-Null

    Write-Output "Connecting to vCenter..."

    # Connect to vCenter Server
    Connect-VIServer -Server $($VC_CONFIG.VC) -User $($VC_CONFIG.VC_USERNAME) -Password $($VC_CONFIG.VC_PASSWORD)

    $vmname = $inputArr[1].trim()
    $vmHost1 = "esxi03.lab.pdotk.com"
    $vmmem = $inputArr[2].trim()
    $vmcpu = $inputArr[3].trim()

    $VMExists = Get-VM -Name $vmname -ErrorAction SilentlyContinue 
    if ($VMExists) {
        Set-VM -VM $VMExists -MemoryMB $vmmem -NumCpu $vmcpu -Confirm:$false #could improve code by catching failures
        Write-Output "VM $vmname modified successfully"
        Invoke-WebRequest $json.response_url -Method 'POST' -Headers @{'Content-Type' = 'application/json; charset=utf-8'} -Body "{'text': 'VM $vmname modified successfully','response_type': 'in_channel'}"
    } else {
        Write-Output "VM $vmname doesn't exist" 
        Invoke-WebRequest $json.response_url -Method 'POST' -Headers @{'Content-Type' = 'application/json; charset=utf-8'} -Body "{'text': 'Could not perform operation: VM $vmname doesn't exists' ,'response_type': 'in_channel'}"
    }

    Disconnect-VIServer * -Confirm:$false
} else {
    Write-Output "Unable to Verify the Key from Gateway"
}