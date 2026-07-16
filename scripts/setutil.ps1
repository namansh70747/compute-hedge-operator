# Forces a position's GPU utilization for the demo, or resumes the random walk.
# Force idle:   pwsh -File scripts/setutil.ps1 -Position batch-render -Util 3
# Resume walk:  pwsh -File scripts/setutil.ps1 -Position batch-render -Util -1
param(
    [string]$Position = "batch-render",
    [double]$Util = 3
)
$ErrorActionPreference = "Stop"
$Ns = "compute-hedge-system"

$pf = Start-Process kubectl -ArgumentList "-n $Ns port-forward svc/gpuexporter 18081:8081" -PassThru -WindowStyle Hidden
try {
    Start-Sleep -Seconds 2
    $r = Invoke-RestMethod -Method Post -Uri "http://localhost:18081/control/$Position`?util=$Util"
    Write-Host "Set $($r.position) forcedUtil=$($r.forcedUtil)"
} finally {
    Stop-Process -Id $pf.Id -Force
}
