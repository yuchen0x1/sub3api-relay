.PHONY: build build-backend test test-backend

# 纯后端个人版，不再编译前端
build: build-backend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 运行后端测试
test: test-backend

test-backend:
	@$(MAKE) -C backend test
