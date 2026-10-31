# ルートからは backend の同名ターゲットを呼ぶ

.DEFAULT_GOAL := help

.PHONY: help test lint integration-test build clean

help: ## コマンド一覧を表示する
	@$(MAKE) -C backend help

test lint integration-test build clean:
	@$(MAKE) -C backend $@
