# Injects a temporary OCPI price spike for a SKU (drives the live demo).
# Usage: pwsh -File scripts/spike.ps1 -Sku H200 -Fraction 0.6
param(
    [string]$Sku = "H200",
    [double]$Fraction = 0.8
)
$ErrorActionPreference = "Stop"
$Ns = "compute-hedge-system"

$pf = Start-Process kubectl -ArgumentList "-n $Ns port-forward svc/mockocpi 18080:8080" -PassThru -WindowStyle Hidden
try {
    Start-Sleep -Seconds 2
    $r = Invoke-RestMethod -Method Post -Uri "http://localhost:18080/spike/$Sku`?fraction=$Fraction"
    Write-Host "Spiked $($r.sku) by fraction $($r.spiked)"
} finally {
    Stop-Process -Id $pf.Id -Force
}
