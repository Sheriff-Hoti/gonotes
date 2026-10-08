# Targets:
#   make venv              # Create/refresh the uv venv
#   make install           # Install laya[serve] into venv
#   make start             # Start Laya HTTP server in background
#   make health            # Check /health endpoint (curl)
#   make test              # Send a test prediction to /v1/systemone
#   make stop              # Stop the server
#   make clean             # Clean venv and cached data

.PHONY: venv install start health test stop clean

# Project root
PROJECT_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
VENV_DIR := $(PROJECT_DIR)/.venv
PID_FILE := /tmp/laya-server.pid

# 1️⃣ Create venv
venv:
	@echo "🔧 Ensuring uv venv with Python 3.12 exists..."
	@if [ ! -d ".venv" ]; then \
		uv venv --python 3.12; \
	else \
		echo "   Venv already exists at .venv"; \
	fi
	@echo "✅ Venv at $(VENV_DIR)"

# 2️⃣ Install laya[serve]
install: venv
	@echo "📦 Installing laya[serve] into venv..."
	uv pip install "laya[serve]"
	@echo "✅ laya[serve] installed"
	@echo "   ▶️  Start with: make start"

# 3️⃣ Start the Laya HTTP server (background, idempotent)
start:
	@echo "🚀 Starting Laya HTTP server in background..."
	@if [ -f "$(PID_FILE)" ] && kill -0 "$(cat $(PID_FILE))" 2>/dev/null; then \
		echo "   Server already running (PID $(cat $(PID_FILE)))"; \
	else \
		@pkill -f "laya-serve" 2>/dev/null || true; \
		sleep 1; \
		LAYA_DEVICE=${LAYA_DEVICE:-cpu} nohup uv run laya-serve > /tmp/laya-server.log 2>&1 & \
		echo $$! > $(PID_FILE); \
		sleep 3; \
		if curl -s http://127.0.0.1:8000/health > /dev/null 2>&1; then \
			echo "   Server ready on http://127.0.0.1:8000"; \
		else \
			echo "   ⚠️  Check /tmp/laya-server.log for status"; \
		fi; \
	fi

# 4️⃣ Health check — just curl, no server start dependency
health:
	@echo "🏥 Checking /health endpoint..."
	@curl -s http://127.0.0.1:8000/health || echo "❌ Server not reachable at 127.0.0.1:8000"

# 5️⃣ Test prediction — just curl, no server start dependency
test:
	@echo "🧪 Sending test prediction to /v1/systemone..."
	@curl -s http://127.0.0.1:8000/v1/systemone \
		-X POST \
		-H 'content-type: application/json' \
		-d '{"state":{"body":"We were billed twice for March. Please refund the duplicate."},"questions":{"department":{"type":"choice","instructions":"Which department should handle this?","criteria":{"billing":"invoices, payments, refunds","technical":"bugs, outages, system errors","other":"everything else"}},"urgency":{"type":"score","instructions":"How urgent is this?","criteria":["not urgent","soon","blocking"]},"churn_risk":{"type":"noul","instructions":"Does the user threaten to cancel or leave?"}}}' | python -m json.tool || echo "❌ Request failed"

# 6️⃣ Stop the running server
stop:
	@echo "🛑 Stopping Laya HTTP server..."
	@if [ -f "$(PID_FILE)" ] && kill -0 "$(cat $(PID_FILE))" 2>/dev/null; then \
		kill "$(cat $(PID_FILE))" 2>/dev/null; \
		rm -f $(PID_FILE); \
		echo "   Server stopped"; \
	else \
		pkill -f "laya-serve" 2>/dev/null || echo "   No server process running"; \
	fi

# 7️⃣ Clean up
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(VENV_DIR)
	@rm -rf .flake8 .mypy_cache .pytest_cache
	@rm -f /tmp/laya-server.log $(PID_FILE)
	@echo "✅ Cleaned"
