IMG ?= compute-hedge-operator:dev
CLUSTER ?= compute-hedge
NS ?= compute-hedge-system

.PHONY: build test vet fmt docker-build kind-create kind-load deploy samples demo teardown

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

docker-build:
	docker build -t $(IMG) .

kind-create:
	kind get clusters | grep -q "^$(CLUSTER)$$" || kind create cluster --name $(CLUSTER) --config hack/kind-config.yaml

kind-load:
	kind load docker-image $(IMG) --name $(CLUSTER)

deploy:
	kubectl apply -f config/crd/computepositions.yaml
	kubectl apply -f config/rbac.yaml
	kubectl apply -f config/manager.yaml
	kubectl apply -f deploy/mockocpi.yaml
	kubectl apply -f deploy/gpuexporter.yaml
	kubectl apply -f deploy/workloads.yaml
	kubectl create configmap grafana-dashboard -n $(NS) \
		--from-file=compute-hedge.json=observability/grafana-dashboard.json \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f observability/prometheus.yaml
	kubectl apply -f observability/grafana.yaml
	kubectl -n $(NS) rollout status deploy/compute-hedge-operator --timeout=120s
	kubectl -n $(NS) rollout status deploy/grafana --timeout=120s

samples:
	kubectl apply -f deploy/samples/computepositions.yaml

demo: docker-build kind-create kind-load deploy samples
	@echo ""
	@echo "Demo is up. Open dashboards with:"
	@echo "  kubectl -n $(NS) port-forward svc/grafana 3000:3000"
	@echo "  kubectl -n $(NS) port-forward svc/prometheus 9090:9090"
	@echo "Grafana: http://localhost:3000  (dashboard: Compute Hedge Operator)"

teardown:
	kind delete cluster --name $(CLUSTER)
