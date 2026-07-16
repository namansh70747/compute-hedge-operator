# Brings up the full demo on a local kind cluster (Windows / PowerShell).
# Usage: pwsh -File scripts/demo.ps1
$ErrorActionPreference = "Stop"

$Img = "compute-hedge-operator:dev"
$Cluster = "compute-hedge"
$Ns = "compute-hedge-system"

# Run from the repo root regardless of where the script is invoked.
Set-Location (Split-Path $PSScriptRoot -Parent)

Write-Host "==> Building image $Img"
docker build -t $Img .

Write-Host "==> Ensuring kind cluster '$Cluster'"
$existing = kind get clusters 2>$null
if ($existing -notcontains $Cluster) {
    kind create cluster --name $Cluster --config hack/kind-config.yaml
}

Write-Host "==> Loading image into kind"
kind load docker-image $Img --name $Cluster

Write-Host "==> Applying manifests"
kubectl apply -f config/crd/computepositions.yaml
kubectl apply -f config/rbac.yaml
kubectl apply -f config/manager.yaml
kubectl apply -f deploy/mockocpi.yaml
kubectl apply -f deploy/gpuexporter.yaml
kubectl apply -f deploy/workloads.yaml

Write-Host "==> Loading Grafana dashboard"
kubectl create configmap grafana-dashboard -n $Ns `
    --from-file=compute-hedge.json=observability/grafana-dashboard.json `
    --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f observability/prometheus.yaml
kubectl apply -f observability/grafana.yaml

Write-Host "==> Waiting for rollouts"
kubectl -n $Ns rollout status deploy/compute-hedge-operator --timeout=120s
kubectl -n $Ns rollout status deploy/grafana --timeout=120s

Write-Host "==> Creating sample positions"
kubectl apply -f deploy/samples/computepositions.yaml

Write-Host ""
Write-Host "Demo is up."
Write-Host "Open the dashboards in two terminals:"
Write-Host "  kubectl -n $Ns port-forward svc/grafana 3000:3000"
Write-Host "  kubectl -n $Ns port-forward svc/prometheus 9090:9090"
Write-Host "Grafana:    http://localhost:3000  (dashboard: Compute Hedge Operator)"
Write-Host "Prometheus: http://localhost:9090/alerts"
Write-Host ""
Write-Host "Watch positions:  kubectl get computepositions -A -w"
