# ==============================================================================
# Configuration Variables
# ==============================================================================

SERVER_NAME  := tap-server
CLI_NAME     := tap-cli
DESKTOP_NAME := tap-desktop

BIN_DIR      := bin
CMD_DIR      := cmd

GO           := go
GOFLAGS      := -ldflags="-w -s"

PORT         ?= 8090
HOST         ?= 127.0.0.1

CYAN         := \033[0;36m
GREEN        := \033[0;32m
RESET        := \033[0m

# ==============================================================================
# Main Rules
# ==============================================================================

# Builds all 3 binaries (Server, CLI, and Desktop GUI) on `make` or `make re`
all: $(BIN_DIR)/$(SERVER_NAME) $(BIN_DIR)/$(CLI_NAME) $(BIN_DIR)/$(DESKTOP_NAME)

$(BIN_DIR)/$(SERVER_NAME): $(CMD_DIR)/server/main.go
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Compiling TAP Central TCP Server...$(RESET)"
	@$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(SERVER_NAME) ./$(CMD_DIR)/server

$(BIN_DIR)/$(CLI_NAME): $(CMD_DIR)/cli/main.go
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Compiling TAP Terminal CLI Client...$(RESET)"
	@$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(CLI_NAME) ./$(CMD_DIR)/cli

$(BIN_DIR)/$(DESKTOP_NAME): $(CMD_DIR)/desktop/main.go
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)Compiling TAP Desktop GUI Client...$(RESET)"
	@$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(DESKTOP_NAME) ./$(CMD_DIR)/desktop
	@echo "$(GREEN)All targets (Server, CLI, Desktop GUI) built in /$(BIN_DIR)!$(RESET)"

desktop: $(BIN_DIR)/$(DESKTOP_NAME)

# ==============================================================================
# Mandatory 42 Subject Targets
# ==============================================================================

install: install-deps

install-deps:
	@$(GO) mod download
	@$(GO) mod tidy

run-server: $(BIN_DIR)/$(SERVER_NAME)
	./$(BIN_DIR)/$(SERVER_NAME) -port $(PORT)

run-client: $(BIN_DIR)/$(CLI_NAME)
	./$(BIN_DIR)/$(CLI_NAME) -host $(HOST) -port $(PORT)

run-client-gui: $(BIN_DIR)/$(DESKTOP_NAME)
	./$(BIN_DIR)/$(DESKTOP_NAME)

lint:
	@$(GO) vet ./...

clean:
	@echo "$(CYAN)Cleaning up binaries...$(RESET)"
	@rm -rf $(BIN_DIR)
	@rm -rf build/bin
	@echo "$(GREEN)Cleanup completed.$(RESET)"

fclean: clean

re: fclean all

.PHONY: all desktop install install-deps run-server run-client run-client-gui lint clean fclean re