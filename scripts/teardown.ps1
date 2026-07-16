# Deletes the local kind cluster.
# Usage: pwsh -File scripts/teardown.ps1
$ErrorActionPreference = "Stop"
kind delete cluster --name compute-hedge
