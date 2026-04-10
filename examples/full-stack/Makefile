OPEN := $(shell command -v xdg-open 2>/dev/null || command -v open 2>/dev/null || echo echo)

.PHONY: help up down logs status populate bench grafana prometheus dashboard clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

up: ## Start all services
	docker compose up -d

down: ## Stop all services and remove volumes
	docker compose down -v

logs: ## Follow container logs
	docker compose logs -f

status: ## Check node health and print service URLs
	@echo "=== Node Health ==="
	@for port in 3901 3902 3903 3904 3905 3906; do \
		printf "node on port $$port: "; \
		curl -s -o /dev/null -w "%{http_code}" http://localhost:$$port/ready 2>/dev/null || echo "unreachable"; \
		echo; \
	done
	@echo ""
	@echo "=== Service URLs ==="
	@echo "Rebost nodes:    http://localhost:3901 .. http://localhost:3906"
	@echo "Dashboards:      http://localhost:3911 .. http://localhost:3916"
	@echo "Grafana:         http://localhost:3000"
	@echo "Prometheus:      http://localhost:9090"
	@echo "Loki:            http://localhost:3100"
	@echo "Tempo:           http://localhost:3200"

C ?= 50
populate: ## Download and upload C random images (default: C=50)
	@mkdir -p .images
	@echo "=== Downloading $(C) images ==="
	@for i in $$(seq 1 $(C)); do \
		W=$$(( (RANDOM % 401) + 400 )); \
		H=$$(( (RANDOM % 401) + 400 )); \
		echo "  [$${i}/$(C)] $${W}x$${H}"; \
		curl -s -L -o .images/photo-$${i}.jpg "https://picsum.photos/$${W}/$${H}"; \
	done
	@echo ""
	@echo "=== Uploading $(C) images ==="
	@for i in $$(seq 1 $(C)); do \
		echo "  [$${i}/$(C)]"; \
		curl -s -X PUT --data-binary @.images/photo-$${i}.jpg "http://localhost:3901/images/photo-$${i}.jpg" > /dev/null; \
	done
	@rm -rf .images
	@echo "Done."

N ?= 500
CONC ?= 10
PORTS := 3901 3902 3903 3904 3905 3906
bench: ## Run ab load test: N=requests CONC=concurrency C=images (defaults: N=500, CONC=10, C=50)
	@echo "=== Benchmarking: $(N) requests, concurrency $(CONC), $(C) images across $(words $(PORTS)) nodes ==="
	@urls=$$(for i in $$(seq 1 $(C)); do \
		for port in $(PORTS); do \
			echo "http://localhost:$${port}/images/photo-$${i}.jpg"; \
		done; \
	done | shuf | head -n $(N)); \
	echo "$$urls" > /tmp/rebost-bench-urls.txt; \
	total=$$(wc -l < /tmp/rebost-bench-urls.txt); \
	ok=0; fail=0; start=$$(date +%s%N); \
	echo "$$urls" | xargs -P $(CONC) -I{} sh -c 'curl -s -o /dev/null -w "%{http_code}\n" "$$1"' _ {} | \
	while read code; do \
		if [ "$$code" = "200" ]; then ok=$$((ok+1)); else fail=$$((fail+1)); fi; \
		done_count=$$((ok+fail)); \
		if [ $$((done_count % 50)) -eq 0 ] || [ "$$done_count" -eq "$$total" ]; then \
			printf "\r  progress: %d/%d (ok=%d fail=%d)" "$$done_count" "$$total" "$$ok" "$$fail"; \
		fi; \
	done; \
	end=$$(date +%s%N); \
	elapsed=$$(( (end - start) / 1000000 )); \
	echo ""; \
	echo ""; \
	echo "=== Results ==="; \
	echo "  Total:       $$total requests"; \
	echo "  Time:        $${elapsed}ms"; \
	if [ "$$elapsed" -gt 0 ]; then \
		rps=$$(( total * 1000 / elapsed )); \
		echo "  Throughput:  ~$${rps} req/s"; \
	fi; \
	rm -f /tmp/rebost-bench-urls.txt

grafana: ## Open Grafana dashboard in browser
	$(OPEN) http://localhost:3000/d/rebost-overview/rebost

prometheus: ## Open Prometheus UI in browser
	$(OPEN) http://localhost:9090

dashboard: ## Open node1 Rebost dashboard in browser
	$(OPEN) http://localhost:3911

clean: ## Remove all containers, volumes, and networks
	docker compose down -v --remove-orphans
